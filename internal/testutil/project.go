// Package testutil provides test data shared by the core packages.
package testutil

import (
	"encoding/json"
	"strconv"

	"github.com/tapirtests/core/model"
)

// ValidProject returns a complete, valid project modelled on the
// "productLifecycle" example from tapir-core-design.md, section 8.4:
//
//	/root
//	  setup:    login (sellerLogin) → sellerToken
//	            login (userLogin)   → userToken
//	  main:
//	    /root/products
//	      main:     productLifecycle
//	                createProductUnauthorized
//	      teardown: cleanupProduct
//
// Every call returns a fresh copy, so tests may modify it freely ("break one
// thing") without affecting each other.
func ValidProject() *model.Project {
	return &model.Project{
		FormatVersion: "1",
		Name:          "shop-e2e",
		BaseURL:       "{{env.BASE_URL}}",
		Spec: model.SpecRef{
			Type:   "swagger2",
			Source: "./openapi.json",
			Hash:   "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
		SecuritySchemes: map[string]*model.SecurityScheme{
			"bearer": {
				Name:      "bearer",
				Type:      model.SecurityAPIKey,
				In:        model.InHeader,
				ParamName: "Authorization",
				Prefix:    "Bearer ",
			},
		},
		Env: []model.EnvVarDecl{
			{Name: "BASE_URL", Description: "API base URL"},
			{Name: "SELLER_USER", Description: "Seller login"},
			{Name: "SELLER_PASS", Secret: true, Description: "Seller password"},
			{Name: "USER_USER", Description: "Buyer login"},
			{Name: "USER_PASS", Secret: true, Description: "Buyer password"},
		},
		Requests:  requests(),
		Scenarios: scenarios(),
		Root:      root(),
	}
}

func requests() map[model.RequestID]*model.RequestDef {
	return map[model.RequestID]*model.RequestDef{
		"login": {
			ID:        "login",
			Operation: model.HTTPOperation{Method: "POST", Path: "/auth/login", Consumes: "application/json"},
			Inputs: []model.InputField{
				{Name: "body.username", Required: true, Schema: schema(`{"type":"string"}`)},
				{Name: "body.password", Required: true, Schema: schema(`{"type":"string"}`)},
			},
			Responses: map[string]model.ResponseDef{
				"200": {Description: "Logged in", Schema: schema(`{"type":"object","required":["token"],"properties":{"token":{"type":"string"}}}`)},
				"401": {Description: "Wrong credentials", Schema: errorSchema()},
			},
		},
		"createProduct": {
			ID:        "createProduct",
			Operation: model.HTTPOperation{Method: "POST", Path: "/products", Consumes: "application/json"},
			Inputs: []model.InputField{
				{Name: "auth.bearer", Required: true},
				{Name: "body.name", Required: true, Schema: schema(`{"type":"string"}`)},
				{Name: "body.price", Required: true, Schema: schema(`{"type":"number"}`)},
				{Name: "body.categoryId", Schema: schema(`{"type":"integer"}`)},
			},
			Responses: map[string]model.ResponseDef{
				"201": {Description: "Created", Schema: productSchema()},
				"400": {Description: "Invalid product", Schema: errorSchema()},
				"401": {Description: "Not authorized", Schema: errorSchema()},
			},
		},
		"getProduct": {
			ID:        "getProduct",
			Operation: model.HTTPOperation{Method: "GET", Path: "/products/{id}"},
			Inputs: []model.InputField{
				{Name: "auth.bearer", Required: true},
				{Name: "path.id", Required: true, Schema: schema(`{"type":"integer"}`)},
			},
			Responses: map[string]model.ResponseDef{
				"200": {Description: "Found", Schema: productSchema()},
				"404": {Description: "Not found", Schema: errorSchema()},
			},
		},
		"deleteProduct": {
			ID:        "deleteProduct",
			Operation: model.HTTPOperation{Method: "DELETE", Path: "/products/{id}"},
			Inputs: []model.InputField{
				{Name: "auth.bearer", Required: true},
				{Name: "path.id", Required: true, Schema: schema(`{"type":"integer"}`)},
			},
			Responses: map[string]model.ResponseDef{
				"204": {Description: "Deleted"},
				"403": {Description: "Not the owner", Schema: errorSchema()},
				"404": {Description: "Not found", Schema: errorSchema()},
			},
		},
	}
}

func scenarios() map[model.ScenarioID]*model.ScenarioDef {
	return map[model.ScenarioID]*model.ScenarioDef{
		"login": {
			ID:          "login",
			Name:        "Log in",
			Description: "Log in and return an access token",
			Inputs: []model.Param{
				{Name: "username"},
				{Name: "password"},
			},
			Outputs: []model.Output{{Name: "token", Value: "{{token}}"}},
			Steps: []model.Step{
				&model.RequestCall{
					ID:        "login",
					RequestID: "login",
					Inputs: map[string]model.Value{
						"body.username": "{{username}}",
						"body.password": "{{password}}",
					},
					Extract: map[string]string{"token": "$.body.token"},
					Expect:  []model.Assertion{status(200)},
				},
			},
		},
		"productLifecycle": {
			ID:          "productLifecycle",
			Name:        "Product lifecycle",
			Description: "Seller creates a product, buyer reads it but cannot delete it, seller deletes it",
			Inputs: []model.Param{
				{Name: "sellerToken"},
				{Name: "userToken"},
			},
			Outputs: []model.Output{{Name: "productId", Value: "{{productId}}"}},
			Steps: []model.Step{
				&model.RequestCall{
					ID:        "create",
					RequestID: "createProduct",
					Inputs: map[string]model.Value{
						"auth.bearer": "{{sellerToken}}",
						"body.name":   "Phone",
						"body.price":  json.Number("412"),
					},
					Extract: map[string]string{"productId": "$.body.id"},
					Expect: []model.Assertion{
						status(201),
						{Target: model.TargetBody, Path: "$.name", Op: model.OpEquals, Value: "Phone"},
					},
				},
				&model.RequestCall{
					ID:        "get-as-user",
					RequestID: "getProduct",
					Inputs: map[string]model.Value{
						"auth.bearer": "{{userToken}}",
						"path.id":     "{{productId}}",
					},
					Expect: []model.Assertion{
						status(200),
						{Target: model.TargetBody, Path: "$.id", Op: model.OpEquals, Value: "{{productId}}"},
						{Target: model.TargetDuration, Op: model.OpLt, Value: json.Number("500")},
					},
				},
				&model.RequestCall{
					ID:        "delete-as-user",
					RequestID: "deleteProduct",
					Inputs: map[string]model.Value{
						"auth.bearer": "{{userToken}}",
						"path.id":     "{{productId}}",
					},
					Expect: []model.Assertion{status(403)},
				},
				&model.RequestCall{
					ID:        "delete-as-seller",
					RequestID: "deleteProduct",
					Inputs: map[string]model.Value{
						"auth.bearer": "{{sellerToken}}",
						"path.id":     "{{productId}}",
					},
					Expect: []model.Assertion{status(204)},
				},
			},
		},
		"createProductUnauthorized": {
			ID:          "createProductUnauthorized",
			Name:        "Create product without auth",
			Description: "An anonymous request must be rejected",
			Steps: []model.Step{
				&model.RequestCall{
					ID:        "create-anonymous",
					RequestID: "createProduct",
					Inputs: map[string]model.Value{
						"auth.bearer": nil, // explicit null: intentionally no authorization
						"body.name":   "Phone",
						"body.price":  json.Number("1"),
					},
					Expect: []model.Assertion{
						status(401),
						{Target: model.TargetBody, Path: "$.message", Op: model.OpExists},
					},
				},
			},
		},
		"cleanupProduct": {
			ID:          "cleanupProduct",
			Name:        "Clean up a product",
			Description: "Delete a product if it still exists",
			Inputs: []model.Param{
				{Name: "sellerToken"},
				{Name: "productId"},
			},
			Steps: []model.Step{
				&model.RequestCall{
					ID:        "delete",
					RequestID: "deleteProduct",
					Inputs: map[string]model.Value{
						"auth.bearer": "{{sellerToken}}",
						"path.id":     "{{productId}}",
					},
					Expect: []model.Assertion{
						{Target: model.TargetStatus, Op: model.OpIn, Value: []any{json.Number("204"), json.Number("404")}},
					},
				},
			},
		},
	}
}

func root() *model.Group {
	return &model.Group{
		Name: "root",
		Setup: []model.ScenarioCall{
			{
				Alias:      "sellerLogin",
				ScenarioID: "login",
				Inputs: map[string]model.Value{
					"username": "{{env.SELLER_USER}}",
					"password": "{{env.SELLER_PASS}}",
				},
				Outputs: map[string]string{"token": "sellerToken"},
			},
			{
				Alias:      "userLogin",
				ScenarioID: "login",
				Inputs: map[string]model.Value{
					"username": "{{env.USER_USER}}",
					"password": "{{env.USER_PASS}}",
				},
				Outputs: map[string]string{"token": "userToken"},
			},
		},
		Main: model.Main{
			Groups: []*model.Group{
				{
					Name:        "products",
					Description: "Product management",
					Main: model.Main{
						Scenarios: []model.ScenarioCall{
							// sellerToken and userToken come from the /root scope.
							{ScenarioID: "productLifecycle"},
							{ScenarioID: "createProductUnauthorized"},
						},
					},
					TearDown: []model.ScenarioCall{
						{
							ScenarioID: "cleanupProduct",
							Inputs:     map[string]model.Value{"productId": "{{productLifecycle.productId}}"},
						},
					},
				},
			},
		},
	}
}

func status(code int) model.Assertion {
	return model.Assertion{
		Target: model.TargetStatus,
		Op:     model.OpEquals,
		Value:  json.Number(strconv.Itoa(code)),
	}
}

func schema(s string) model.Schema { return model.Schema(s) }

// Schemas are built by functions, not shared variables: Schema is a byte
// slice, and a shared one modified by a test would leak into other tests.

func errorSchema() model.Schema {
	return schema(`{"type":"object","properties":{"message":{"type":"string"}}}`)
}

func productSchema() model.Schema {
	return schema(`{"type":"object","required":["id","name","price"],"properties":{"id":{"type":"integer"},"name":{"type":"string"},"price":{"type":"number"},"categoryId":{"type":"integer"}}}`)
}
