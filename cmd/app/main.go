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
	"github.com/ituoga/toolbox"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	datastar "github.com/starfederation/datastar/sdk/go"
)

func main() {
	ctx := context.Background()

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

	router := chi.NewRouter()

	router.Get("/products", func(w http.ResponseWriter, r *http.Request) {
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

	router.Get("/products/seed", func(w http.ResponseWriter, r *http.Request) {
		for i := range 10 {
			bee.PublishCommand(ctx, &gen.CommandEnvelope{
				Aggregate:   "product",
				AggregateId: fmt.Sprintf("prod-%d", time.Now().UnixNano()),
				CommandType: "create",
				Payload:     []byte(fmt.Sprintf(`{"name":"I: %d then - Product %d","price":10.0}`, i, time.Now().UnixNano())),
			}, nil)
		}
		// <-msg
		http.Redirect(w, r, "/products", http.StatusSeeOther)
	})

	router.Get("/cart", func(w http.ResponseWriter, r *http.Request) {
		pages.Cart(collection.Cart{}).Render(r.Context(), w)
	})

	router.Post("/cart/remove/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Missing product ID", http.StatusBadRequest)
			return
		}

		lctx := bee.WithJetStream(r.Context(), js)
		lctx = bee.WithNats(lctx, nc)

		err := bee.PublishCommand(lctx, &gen.CommandEnvelope{
			Aggregate:   "cart",
			AggregateId: "cart-1",
			CommandType: "remove_item",
			Payload:     []byte(fmt.Sprintf(`{"ProductID":"%s"}`, id)),
		}, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to remove item: %v", err), http.StatusInternalServerError)
			return
		}

		datastar.NewSSE(w, r)
	})

	router.Post("/cart/add-product-id/{id}", func(w http.ResponseWriter, r *http.Request) {
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

		datastar.NewSSE(w, r)
	})
	router.Post("/cart/add-product", func(w http.ResponseWriter, r *http.Request) {
		lctx := bee.WithJetStream(r.Context(), js)
		lctx = bee.WithNats(lctx, nc)
		bee.PublishCommand(lctx, &gen.CommandEnvelope{
			Aggregate:   "cart",
			AggregateId: "cart-1",
			CommandType: "add_item",
			Payload:     []byte(fmt.Sprintf(`{"product":{"ID":"prod-%d","name":"Product %d","price":%d}}`, time.Now().UnixNano(), time.Now().UnixNano(), rand.Intn(100))),
		}, nil)
		datastar.NewSSE(w, r)
	})

	router.Get("/cart/count", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		sse := datastar.NewSSE(w, r)
		lctx := bee.WithJetStream(r.Context(), js)
		lctx = bee.WithNats(lctx, nc)

		go func() {
			aggProduct := &UpdateProductLiveProjection{}
			updatesProducts := bee.ReplayAndSubscribe(lctx, aggProduct, ro.WithAggreate("product"), ro.WithAggregateID("*"))

			for {
				select {
				case <-lctx.Done():
					log.Println("Context done, stopping product updates")
					return
				case update := <-updatesProducts:
					if update == nil {
						log.Println("No updates received, stopping product updates")
						return
					}
					if update.err != nil {
						continue
					}

					var products []shopping.Product
					db.ReadTX(r.Context(), func(tx *rwdb.Tx) error {
						if err := tx.Model(shopping.Product{}).Find(&products).Error; err != nil {
							log.Printf("Failed to fetch products: %v", err)
							return err
						}
						return nil
					})
					sse.MergeFragmentTempl(pages.ProductItem(collection.Product{
						Products: products,
					}))
				}
			}
		}()

		agg := &CartCounterLiveProjection{}
		updates := bee.ReplayAndSubscribe(lctx, agg, ro.WithAggreate("cart"), ro.WithAggregateID("cart-1"))
		for {
			select {
			case <-lctx.Done():
				log.Println("Context done, stopping cart count updates")
				return
			case update := <-updates:
				if update == nil {
					log.Println("No updates received, stopping cart count updates")
					return
				}
				sse.MergeFragmentTempl(pages.CartCount(agg.Count))
			}
		}
	})

	router.Get("/cart/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		sse := datastar.NewSSE(w, r)
		lctx := bee.WithJetStream(r.Context(), js)
		lctx = bee.WithNats(lctx, nc)

		go func() {
			for range 5 {

				time.Sleep(1 * time.Second)
			}
		}()
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

type UpdateProductLiveProjection struct {
	err error
}

func (p *UpdateProductLiveProjection) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}
	p.err = nil // Reset error on each event
	switch event.(type) {
	case *shopping.ProductCreated:
		return nil // Ignore product creation event
	default:
		p.err = errors.New("unsupported event type for UpdateProductLiveProjection")
	}
	return nil
}

type CartCounterLiveProjection struct {
	Count int `json:"count"` // Count of items in the cart
}

// ApplyEvent applies an event to the CartCounterLiveProjection
func (c *CartCounterLiveProjection) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}
	switch event.(type) {
	case *shopping.CartItemAdded:
		c.Count++
	case *shopping.CartItemRemoved:
		c.Count--
	default:
		return nil // Ignore other event types
	}
	return nil
}

type CartProjection struct {
	Items []shopping.Product `json:"history"` // History of events applied to the aggregate
}

func (a *CartProjection) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}
	switch event := event.(type) {
	case *shopping.CartItemAdded:
		a.Items = append(a.Items, shopping.Product{
			ID:    event.Product.ID,
			Name:  event.Product.Name,
			Price: event.Product.Price,
		})
	case *shopping.CartItemRemoved:
		for i, p := range a.Items {
			if p.ID == event.ProductID {
				a.Items = append(a.Items[:i], a.Items[i+1:]...)
				break
			}
		}
	default:
		return nil // Ignore other event types
	}
	return nil
}
