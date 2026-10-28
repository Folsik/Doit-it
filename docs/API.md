# API notes

- Base path: `/v1`
- JSON request/response bodies
- Auth header: `Authorization: Bearer <access_token>`
- Error envelope: `{ "error": { "code": "...", "message": "..." } }`
- Money: integer minor units (`amount_minor`), currency code such as `EUR`
- Timestamps: ISO 8601 / RFC 3339 UTC

The API is a starter contract and should be formalized in OpenAPI before mobile implementation.
