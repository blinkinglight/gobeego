package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/co"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/po"
	"github.com/blinkinglight/bee/ro"
	"github.com/blinkinglight/gobeego/apps/shopping"
	"github.com/blinkinglight/gobeego/pkg/appctx"
	"github.com/blinkinglight/gobeego/pkg/collection"
	"github.com/blinkinglight/gobeego/pkg/rwdb"
	"github.com/blinkinglight/gobeego/web/pages"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ituoga/toolbox"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	datastar "github.com/starfederation/datastar/sdk/go"
	"gorm.io/gorm"
)

func main() {
	ctx := context.Background()
	// datastar.WithGzip(datastar.WithGzipLevel(9))
	datastar.WithBrotli()

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	fmt.Println(exPath)

	fp, _ := toolbox.FreePort()
	ns, err := embeddednats.New(ctx, embeddednats.WithNATSServerOptions(&server.Options{
		JetStream: true,
		StoreDir:  exPath + "/data",
		Port:      fp,
	}), embeddednats.WithShouldClearData(true), embeddednats.WithDirectory(exPath+"/data"))
	if err != nil {
		panic(err)
	}
	log.Printf("NATS server started on port %d", fp)

	ns.WaitForServer()
	nc, err := ns.Client()
	// nc, err := nats.Connect("nats://localhost:4222", nats.Name("gobeego shopping app"))
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		panic(err)
	}

	js.AddStream(&nats.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"events.>"},
		Storage:  nats.FileStorage,
	})

	db := rwdb.Open("./data/shopping.db")
	ctx = appctx.WithDB(ctx, db)

	db.WriteTX(ctx, func(tx *rwdb.Tx) error {
		if err := tx.AutoMigrate(&shopping.Cart{}, &shopping.Product{}); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
		return nil
	})

	ctx = bee.WithNats(ctx, nc)
	ctx = bee.WithJetStream(ctx, js)

	go bee.Command(ctx, &shopping.CartService{Ctx: ctx}, co.WithAggreate("cart"))
	go bee.Command(ctx, &shopping.UserService{Ctx: ctx}, co.WithAggreate("user"))

	go bee.Command(ctx, &ProductService{Ctx: ctx}, co.WithAggreate("product"))
	go bee.Project(ctx, &ProductProjection{Ctx: ctx}, po.WithAggreate("product"))

	chi.RegisterMethod("DS_GET")
	chi.RegisterMethod("DS_POST")

	router := chi.NewRouter()

	router.Use(OverrideMethodByHeader)

	router.Get("/products", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		var products []shopping.Product
		err := db.ReadTX(r.Context(), func(tx *rwdb.Tx) error {
			return tx.Model(shopping.Product{}).Find(&products).Error
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch products: %v", err), http.StatusInternalServerError)
			return
		}
		pages.Products(collection.Product{
			Products: products}).Render(r.Context(), w)
	})

	router.MethodFunc("DS_GET", "/products/seed", func(w http.ResponseWriter, r *http.Request) {
		for i := range 10 {
			bee.PublishCommand(ctx, &gen.CommandEnvelope{
				Aggregate:   "product",
				AggregateId: fmt.Sprintf("prod-%d", time.Now().UnixNano()),
				CommandType: "create",
				Payload:     []byte(fmt.Sprintf(`{"name":"I: %d then - Product %d","price":10.0}`, i, time.Now().UnixNano())),
			}, nil)
		}
		w.WriteHeader(200)
		datastar.NewSSE(w, r)
	})

	router.MethodFunc("DS_GET", "/product/{id}/live", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Missing product ID", http.StatusBadRequest)
			return
		}

		w.WriteHeader(200)
		sse := datastar.NewSSE(w, r)
		lctx := bee.WithJetStream(r.Context(), js)
		lctx = bee.WithNats(lctx, nc)

		agg := &ProductLiveView{}
		updates := bee.ReplayAndSubscribe(lctx, agg, ro.WithAggreate("product"), ro.WithAggregateID(id))
		for {
			select {
			case <-lctx.Done():
				log.Println("Context done, stopping product updates")
				return
			case update := <-updates:
				if update == nil {
					log.Println("No updates received, stopping product updates")
					return
				}

				sse.MergeFragmentTempl(pages.ProductSingleItem(update.Product))
			}
		}
	})

	router.Get("/product/create", func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		if id == "" {
			http.Error(w, "Missing product ID", http.StatusBadRequest)
			return
		}
		bee.PublishCommand(ctx, &gen.CommandEnvelope{
			Aggregate:   "product",
			AggregateId: id,
			CommandType: "create",
		}, &shopping.Product{
			ID:    id,
			Name:  fmt.Sprintf("Product %s", id),
			Price: 10.0,
		})
		http.Redirect(w, r, fmt.Sprintf("/product/%s", id), http.StatusSeeOther)
	})

	router.Get("/product/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Missing product ID", http.StatusBadRequest)
			return
		}

		var product shopping.Product
		err := db.ReadTX(r.Context(), func(tx *rwdb.Tx) error {
			return tx.Model(shopping.Product{}).Where("id = ?", id).First(&product).Error
		})
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, fmt.Sprintf("Failed to fetch product: %v", err), http.StatusInternalServerError)
			return
		}
		product.ID = id // Ensure product ID is set
		w.WriteHeader(200)
		pages.ProductSingle(product).Render(r.Context(), w)
	})

	router.Get("/cart", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		pages.Cart(collection.Cart{}).Render(r.Context(), w)
	})

	router.MethodFunc("DS_POST", "/cart/remove/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Missing product ID", http.StatusBadRequest)
			return
		}

		lctx := bee.WithJetStream(r.Context(), js)
		lctx = bee.WithNats(lctx, nc)

		var product shopping.Product
		err := db.ReadTX(r.Context(), func(tx *rwdb.Tx) error {
			return tx.Model(shopping.Product{}).Where("id = ?", id).First(&product).Error
		})
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, fmt.Sprintf("Failed to fetch product: %v", err), http.StatusInternalServerError)
			return
		}

		cir := shopping.CartItemRemove{
			ProductID: id,
		}

		err = bee.PublishCommand(lctx, &gen.CommandEnvelope{
			Aggregate:   "cart",
			AggregateId: "cart-1",
			CommandType: "remove_item",
		}, cir)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to remove item: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(200)
		datastar.NewSSE(w, r)
	})

	router.MethodFunc("DS_POST", "/cart/add-product-id/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Missing product ID", http.StatusBadRequest)
			return
		}

		lctx := bee.WithJetStream(r.Context(), js)
		lctx = bee.WithNats(lctx, nc)

		product := shopping.Product{}
		err := db.ReadTX(r.Context(), func(tx *rwdb.Tx) error {
			return tx.Model(shopping.Product{}).Where("id = ?", id).First(&product).Error
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch product: %v", err), http.StatusInternalServerError)
			return
		}

		err = bee.PublishCommand(lctx, &gen.CommandEnvelope{
			Aggregate:   "cart",
			AggregateId: "cart-1",
			CommandType: "add_item",
			Payload:     []byte(fmt.Sprintf(`{"product":{"ID":"%s","name":"%s","price":%.02f}}`, product.ID, product.Name, product.Price)),
		}, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to add item: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(200)
		datastar.NewSSE(w, r)
	})
	router.MethodFunc("DS_POST", "/cart/add-product", func(w http.ResponseWriter, r *http.Request) {
		lctx := bee.WithJetStream(r.Context(), js)
		lctx = bee.WithNats(lctx, nc)
		bee.PublishCommand(lctx, &gen.CommandEnvelope{
			Aggregate:   "cart",
			AggregateId: "cart-1",
			CommandType: "add_item",
			Payload:     []byte(fmt.Sprintf(`{"product":{"ID":"prod-%d","name":"Product %d","price":%d}}`, time.Now().UnixNano(), time.Now().UnixNano(), rand.Intn(100))),
		}, nil)
		w.WriteHeader(200)
		datastar.NewSSE(w, r)
	})

	router.MethodFunc("DS_GET", "/cart/count", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		sse := datastar.NewSSE(w, r)
		lctx := bee.WithJetStream(r.Context(), js)
		lctx = bee.WithNats(lctx, nc)

		aggProduct := &UpdateProductLiveProjection{}
		updatesProducts := bee.ReplayAndSubscribe(lctx, aggProduct, ro.WithAggreate("product"), ro.WithAggregateID("*"))

		agg := &CartCounterLiveProjection{}
		updates := bee.ReplayAndSubscribe(lctx, agg, ro.WithAggreate("cart"), ro.WithAggregateID("cart-1"))
		go func() {
			for {
				select {
				case <-lctx.Done():
					log.Println("Context done, stopping cart count updates")
					return
				case update1 := <-updatesProducts:
					if update1 == nil {
						log.Println("No updates received, stopping product updates")
						return
					}
					if update1.err != nil {
						log.Printf("Error in UpdateProductLiveProjection: %v", update1.err)
						continue
					}

					// db.ReadTX(r.Context(), func(tx *rwdb.Tx) error {
					// 	if err := tx.Model(shopping.Product{}).Find(&products).Error; err != nil {
					// 		log.Printf("Failed to fetch products: %v", err)
					// 		return err
					// 	}
					// 	return nil
					// })
					sse.MergeFragmentTempl(pages.ProductItem(collection.Product{
						Products: update1.Products,
					}))
				case update := <-updates:
					if update == nil {
						log.Println("No updates received, stopping cart count updates")
						return
					}
					sse.MergeFragmentTempl(pages.CartCount(agg.Count, agg.Total))
				}
			}
		}()
		<-lctx.Done() // Wait for context to be done
	})

	router.MethodFunc("DS_GET", "/cart/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		sse := datastar.NewSSE(w, r)
		lctx := bee.WithJetStream(r.Context(), js)
		lctx = bee.WithNats(lctx, nc)

		agg := &CartProjection{}
		updates := bee.ReplayAndSubscribe(lctx, agg, ro.WithAggreate("cart"), ro.WithAggregateID("cart-1"))
		for {
			select {
			case <-lctx.Done():
				log.Println("Context done, stopping live updates")
				return
			case update := <-updates:
				if update == nil {
					log.Println("No updates received, stopping live updates")
					return
				}
				sse.MergeFragmentTempl(pages.CartItems(collection.Cart{
					Products: update.Items,
				}))
			}
		}

	})

	bee.PublishCommand(ctx, &gen.CommandEnvelope{
		Aggregate:   "cart",
		AggregateId: "cart-1",
		CommandType: "create",
		Payload:     []byte(`{"items":[],"total":0,"discount":0}`),
	}, nil)

	log.Printf("Starting server on http://localhost:4321")
	log.Fatal(http.ListenAndServe(":4321", router))

}

func OverrideMethodByHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("datastar-request") != "" {
			r.Method = "DS_" + r.Method
		}
		next.ServeHTTP(w, r)
	})
}
