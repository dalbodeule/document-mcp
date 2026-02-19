package httpapi

// NOTE: This is a minimal OpenAPI spec to bootstrap Swagger UI.
// Extend as endpoints evolve.

func OpenAPISpecJSON() []byte {
	return []byte(openapiJSON)
}

const openapiJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "document-mcp API",
    "version": "0.1.0"
  },
  "paths": {
    "/health": {
      "get": {
        "summary": "Health check",
        "responses": {"200": {"description": "OK"}}
      }
    },
    "/auth/register": {
      "post": {
        "summary": "Register user",
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/RegisterRequest"}}}},
        "responses": {"201": {"description": "Created"}}
      }
    },
    "/auth/login": {
      "post": {
        "summary": "Login",
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/LoginRequest"}}}},
        "responses": {"200": {"description": "OK", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Tokens"}}}}}
      }
    },
    "/auth/refresh": {
      "post": {
        "summary": "Refresh tokens",
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/RefreshRequest"}}}},
        "responses": {"200": {"description": "OK", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Tokens"}}}}}
      }
    },
    "/auth/logout": {
      "post": {
        "summary": "Logout",
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/LogoutRequest"}}}},
        "responses": {"204": {"description": "No Content"}}
      }
    },
    "/groups": {
      "post": {
        "summary": "Create group",
        "security": [{"bearerAuth": []}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/CreateGroupRequest"}}}},
        "responses": {"201": {"description": "Created"}}
      }
    },
    "/me/groups": {
      "get": {
        "summary": "List my groups",
        "security": [{"bearerAuth": []}],
        "responses": {"200": {"description": "OK"}}
      }
    },
    "/documents": {
      "post": {
        "summary": "Create document (stores embedding + aliases)",
        "security": [{"bearerAuth": []}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/CreateDocumentRequest"}}}},
        "responses": {"201": {"description": "Created"}}
      }
    },
    "/search": {
      "post": {
        "summary": "Vector search",
        "security": [{"bearerAuth": []}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/SearchRequest"}}}},
        "responses": {"200": {"description": "OK"}}
      }
    }
  },
  "components": {
    "securitySchemes": {
      "bearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT"
      }
    },
    "schemas": {
      "RegisterRequest": {
        "type": "object",
        "required": ["email", "password", "name"],
        "properties": {
          "email": {"type": "string", "format": "email"},
          "password": {"type": "string", "minLength": 8},
          "name": {"type": "string"}
        }
      },
      "LoginRequest": {
        "type": "object",
        "required": ["email", "password"],
        "properties": {
          "email": {"type": "string", "format": "email"},
          "password": {"type": "string"}
        }
      },
      "RefreshRequest": {
        "type": "object",
        "required": ["refresh_token"],
        "properties": {
          "refresh_token": {"type": "string"}
        }
      },
      "LogoutRequest": {
        "type": "object",
        "required": ["refresh_token"],
        "properties": {
          "refresh_token": {"type": "string"}
        }
      },
      "Tokens": {
        "type": "object",
        "required": ["access_token", "refresh_token"],
        "properties": {
          "access_token": {"type": "string"},
          "refresh_token": {"type": "string"}
        }
      },
      "CreateGroupRequest": {
        "type": "object",
        "required": ["name"],
        "properties": {
          "name": {"type": "string"}
        }
      },
      "CreateDocumentRequest": {
        "type": "object",
        "required": ["year", "depth1", "depth2", "title"],
        "properties": {
          "year": {"type": "integer"},
          "depth1": {"type": "string"},
          "depth2": {"type": "string"},
          "title": {"type": "string"},
          "content": {"type": "string"},
          "aliases": {"type": "array", "items": {"type": "string"}}
        }
      },
      "SearchRequest": {
        "type": "object",
        "required": ["query"],
        "properties": {
          "query": {"type": "string"},
          "year": {"type": "integer"},
          "depth1": {"type": "string"},
          "depth2": {"type": "string"},
          "alias": {"type": "string"},
          "limit": {"type": "integer", "minimum": 1, "maximum": 100}
        }
      }
    }
  }
}`
