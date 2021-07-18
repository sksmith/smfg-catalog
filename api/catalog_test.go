package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/sksmith/smfg-catalog/api"
	"github.com/sksmith/smfg-catalog/core"
	"github.com/sksmith/smfg-catalog/core/catalog"
	"github.com/sksmith/smfg-catalog/db"
	"github.com/sksmith/smfg-catalog/queue"
)

func configureServer(s catalog.Service) *httptest.Server {
	r := chi.NewRouter()

	catalogApi := api.NewCatalogApi(s)
	catalogApi.ConfigureRouter(r)

	return httptest.NewServer(r)
}

func TestCreate(t *testing.T) {
	mockRepo := db.NewMockRepo()
	mockQueue := queue.NewMockQueue()

	tp := testProducts[0]

	mockRepo.SaveProductFunc = func(ctx context.Context, product catalog.Product, tx ...core.Transaction) error {
		if product.Name != tp.Name {
			t.Errorf("name got=%s want=%s", product.Name, tp.Name)
		}
		if product.Sku != tp.Sku {
			t.Errorf("sku got=%s want=%s", product.Sku, tp.Sku)
		}
		if product.Upc != tp.Upc {
			t.Errorf("upc got=%s want=%s", product.Upc, tp.Upc)
		}
		return nil
	}

	service := catalog.NewService(mockRepo, mockQueue, "product.fanout")
	ts := configureServer(service)
	defer ts.Close()

	data, err := json.Marshal(tp)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(ts.URL+"/v1", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%s", body)
}

var testProducts = []catalog.Product{
	{
		Sku:  "sku1",
		Upc:  "upc1",
		Name: "name1",
	},
	{
		Sku:  "sku2",
		Upc:  "upc2",
		Name: "name2",
	},
	{
		Sku:  "sku3",
		Upc:  "upc3",
		Name: "name3",
	},
}
