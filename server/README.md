# Mattermost Legal Hold Plugin REST API Documentation

The Mattermost Legal Hold Plugin provides RESTful APIs for managing legal holds and testing Amazon S3 connections. Below are the details of the available endpoints including JSON schema for request bodies and responses:

## Authentication
All endpoints require authentication. Each request must include a valid Mattermost user token in the `Authorization` header.
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

## Endpoints

### 1. List Legal Holds
Retrieve a list of all legal holds.

- **URL:** `/api/v1/legalholds`
- **Method:** `GET`
- **Description:** Returns a list of all legal holds.
- **Response:**
  - `200 OK`: Success
  - JSON array of legal hold objects.

#### Response Schema:
```json
[
  {
    "id": "string",
    "name": "string",
    "description": "string",
    "created_at": "string (ISO 8601 date-time)",
    "status": "string",
    ...
  }
]
```

**Example Request:**
```http
GET /api/v1/legalholds
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Example Response:**
```json
[
  {
    "id": "legalhold1",
    "name": "Legal Hold 1",
    "description": "Description of Legal Hold 1",
    "created_at": "2023-01-01T12:00:00Z",
    "status": "active"
  }
]
```

### 2. Create a Legal Hold
Create a new legal hold.

- **URL:** `/api/v1/legalholds`
- **Method:** `POST`
- **Description:** Creates a new legal hold with the specified details.
- **Request Body:**
  - JSON object with legal hold details.
- **Response:**
  - `201 Created`: Success
  - JSON object of the created legal hold.

#### Request Schema:
```json
{
  "name": "string",
  "description": "string",
  ...
}
```

#### Response Schema:
```json
{
  "id": "string",
  "name": "string",
  "description": "string",
  "created_at": "string (ISO 8601 date-time)",
  "status": "string",
  ...
}
```

**Example Request:**
```http
POST /api/v1/legalholds
Authorization: Bearer YOUR_ACCESS_TOKEN
Content-Type: application/json

{
  "name": "New Legal Hold",
  "description": "Description of the new legal hold"
}
```

**Example Response:**
```json
{
  "id": "new-legalhold-id",
  "name": "New Legal Hold",
  "description": "Description of the new legal hold",
  "created_at": "2023-02-01T12:00:00Z",
  "status": "active"
}
```

### 3. Release a Legal Hold
Release an existing legal hold.

- **URL:** `/api/v1/legalholds/{legalhold_id}/release`
- **Method:** `POST`
- **Description:** Releases the specified legal hold.
- **Path Parameters:**
  - `legalhold_id`: The ID of the legal hold to release.
- **Response:**
  - `200 OK`: Success

**Example Request:**
```http
POST /api/v1/legalholds/legalhold1/release
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Example Response:**
```http
200 OK
```

### 4. Update a Legal Hold
Update the details of an existing legal hold.

- **URL:** `/api/v1/legalholds/{legalhold_id}`
- **Method:** `PUT`
- **Description:** Updates the specified legal hold with new details.
- **Path Parameters:**
  - `legalhold_id`: The ID of the legal hold to update.
- **Request Body:**
  - JSON object with updated legal hold details.
- **Response:**
  - `200 OK`: Success

#### Request Schema:
```json
{
  "name": "string",
  "description": "string",
  ...
}
```

#### Response Schema:
```json
{
  "id": "string",
  "name": "string",
  "description": "string",
  "created_at": "string (ISO 8601 date-time)",
  "status": "string",
  ...
}
```

**Example Request:**
```http
PUT /api/v1/legalholds/legalhold1
Authorization: Bearer YOUR_ACCESS_TOKEN
Content-Type: application/json

{
  "name": "Updated Legal Hold Name",
  "description": "Updated description of the legal hold"
}
```

**Example Response:**
```json
{
  "id": "legalhold1",
  "name": "Updated Legal Hold Name",
  "description": "Updated description of the legal hold",
  "created_at": "2023-01-01T12:00:00Z",
  "status": "active"
}
```

### 5. Download a Legal Hold
Download the details of an existing legal hold.

- **URL:** `/api/v1/legalholds/{legalhold_id}/download`
- **Method:** `GET`
- **Description:** Downloads the specified legal hold details.
- **Path Parameters:**
  - `legalhold_id`: The ID of the legal hold to download.
- **Response:**
  - `200 OK`: Success
  - JSON object of the legal hold details.

#### Response Schema:
```json
{
  "id": "string",
  "name": "string",
  "description": "string",
  "created_at": "string (ISO 8601 date-time)",
  "status": "string",
  ...
}
```

**Example Request:**
```http
GET /api/v1/legalholds/legalhold1/download
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Example Response:**
```json
{
  "id": "legalhold1",
  "name": "Legal Hold 1",
  "description": "Description of Legal Hold 1",
  "created_at": "2023-01-01T12:00:00Z",
  "status": "active"
}
```

### 6. Test Amazon S3 Connection
Test the Amazon S3 connection configuration.

- **URL:** `/api/v1/test_amazon_s3_connection`
- **Method:** `POST`
- **Description:** Tests the configured Amazon S3 connection.
- **Response:**
  - `200 OK`: Success if connection is valid.
  - `400 Bad Request`: Invalid connection details.

#### Response Schema:
```json
{
  "success": "boolean",
  "message": "string"
}
```

**Example Request:**
```http
POST /api/v1/test_amazon_s3_connection
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Example Response:**
```json
{
  "success": true,
  "message": "Amazon S3 connection is valid"
}
```

---

This documentation covers the functionalities of APIs for managing legal holds and testing Amazon S3 connections in the Mattermost Legal Hold Plugin. Ensure that appropriate authentication tokens are provided in the request headers to successfully interact with these endpoints.