package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
	"github.com/sksmith/smfg-catalog/core"
	"github.com/sksmith/smfg-catalog/core/catalog"
)

type CatalogApi struct {
	service catalog.Service
}

func NewCatalogApi(service catalog.Service) *CatalogApi {
	return &CatalogApi{service: service}
}

const (
	CtxKeyProduct     CtxKey = "product"
	CtxKeyReservation CtxKey = "reservation"
)

func (a *CatalogApi) ConfigureRouter(r chi.Router) {
	r.Route("/v1", func(r chi.Router) {
		r.Put("/", a.Create)

		r.Route("/{sku}", func(r chi.Router) {
			r.Get("/", a.GetProduct)
		})
	})
}

type ProductResponse struct {
	catalog.Product
}

func NewProductResponse(product catalog.Product) *ProductResponse {
	resp := &ProductResponse{Product: product}
	return resp
}

func (rd *ProductResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	// Pre-processing before a response is marshalled and sent across the wire
	return nil
}

func (a *CatalogApi) Create(w http.ResponseWriter, r *http.Request) {
	data := &CreateProductRequest{}
	if err := render.Bind(r, data); err != nil {
		Render(w, r, ErrInvalidRequest(err))
		return
	}

	if err := a.service.CreateProduct(r.Context(), *data.Product); err != nil {
		log.Err(err).Send()
		Render(w, r, ErrInternalServer)
		return
	}

	render.Status(r, http.StatusCreated)
	Render(w, r, NewProductResponse(*data.Product))
}

func (a *CatalogApi) GetProduct(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")
	if sku == "" {
		Render(w, r, ErrInvalidRequest(errors.New("sku is required")))
		return
	}

	product, err := a.service.GetProduct(r.Context(), sku)

	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			Render(w, r, ErrNotFound)
		} else {
			log.Error().Err(err).Str("sku", sku).Msg("error acquiring product")
			Render(w, r, ErrInternalServer)
		}
		return
	}
	Render(w, r, NewProductResponse(product))
}

type CreateProductRequest struct {
	*catalog.Product
}

func (p *CreateProductRequest) Bind(_ *http.Request) error {
	if p.Upc == "" || p.Name == "" || p.Sku == "" {
		return errors.New("missing required field(s)")
	}

	return nil
}

func Render(w http.ResponseWriter, r *http.Request, rnd render.Renderer) {
	if err := render.Render(w, r, rnd); err != nil {
		log.Warn().Err(err).Msg("failed to render")
	}
}

func RenderList(w http.ResponseWriter, r *http.Request, l []render.Renderer) {
	if err := render.RenderList(w, r, l); err != nil {
		log.Warn().Err(err).Msg("failed to render")
	}
}
