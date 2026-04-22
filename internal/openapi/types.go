// Package openapi builds the OpenAPI 3.0.3 spec programmatically.
// No annotation magic — just Go structs that mirror the actual handlers.
// Update this file when you add or change routes.
package openapi

// Spec is the root OpenAPI 3.0 document.
type Spec struct {
	OpenAPI    string                `json:"openapi"`
	Info       Info                  `json:"info"`
	Servers    []Server              `json:"servers"`
	Paths      map[string]PathItem   `json:"paths"`
	Components Components            `json:"components"`
	Tags       []Tag                 `json:"tags"`
}

type Info struct {
	Title       string  `json:"title"`
	Version     string  `json:"version"`
	Description string  `json:"description"`
	License     License `json:"license"`
}

type License struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

type Operation struct {
	OperationID string              `json:"operationId"`
	Summary     string              `json:"summary"`
	Tags        []string            `json:"tags"`
	Security    []SecurityReq       `json:"security,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type SecurityReq map[string][]string

type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // query | path | header
	Required    bool    `json:"required"`
	Description string  `json:"description,omitempty"`
	Schema      Schema  `json:"schema"`
}

type RequestBody struct {
	Required bool               `json:"required"`
	Content  map[string]MediaType `json:"content"`
}

type MediaType struct {
	Schema Schema `json:"schema"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type Components struct {
	Schemas         map[string]Schema         `json:"schemas"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes"`
}

type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Description  string `json:"description,omitempty"`
}

type Schema struct {
	Type        string            `json:"type,omitempty"`
	Format      string            `json:"format,omitempty"`
	Description string            `json:"description,omitempty"`
	Properties  map[string]Schema `json:"properties,omitempty"`
	Items       *Schema           `json:"items,omitempty"`
	Required    []string          `json:"required,omitempty"`
	Ref         string            `json:"$ref,omitempty"`
	Enum        []string          `json:"enum,omitempty"`
	Example     any               `json:"example,omitempty"`
}

// Helpers

func ref(name string) Schema { return Schema{Ref: "#/components/schemas/" + name} }
func arr(s Schema) Schema    { return Schema{Type: "array", Items: &s} }
func str(desc string) Schema { return Schema{Type: "string", Description: desc} }
func uuid_(desc string) Schema {
	return Schema{Type: "string", Format: "uuid", Description: desc}
}
func ts(desc string) Schema {
	return Schema{Type: "string", Format: "date-time", Description: desc}
}
func bearer() []SecurityReq { return []SecurityReq{{"bearerAuth": []string{}}} }

func jsonBody(schema Schema) *RequestBody {
	return &RequestBody{
		Required: true,
		Content:  map[string]MediaType{"application/json": {Schema: schema}},
	}
}

func jsonResponse(desc string, schema Schema) Response {
	return Response{
		Description: desc,
		Content: map[string]MediaType{
			"application/json": {
				Schema: Schema{
					Type: "object",
					Properties: map[string]Schema{
						"data": schema,
					},
				},
			},
		},
	}
}

func errResponse(desc string) Response {
	return Response{
		Description: desc,
		Content: map[string]MediaType{
			"application/json": {
				Schema: Schema{
					Type:       "object",
					Properties: map[string]Schema{"error": str(desc)},
				},
			},
		},
	}
}

func queryParam(name, desc string, required bool) Parameter {
	return Parameter{Name: name, In: "query", Required: required, Description: desc, Schema: str("")}
}

func pathParam(name, desc string) Parameter {
	return Parameter{Name: name, In: "path", Required: true, Description: desc, Schema: uuid_("")}
}
