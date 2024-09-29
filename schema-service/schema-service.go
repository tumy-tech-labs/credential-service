package main

import (
	"database/sql"
	"encoding/json"

	_ "github.com/lib/pq"
)

type Schema struct {
	ID              string     `json:"id"`
	OrganizationDID string     `json:"organization_did"`
	SchemaName      string     `json:"schema_name"`
	Properties      []Property `json:"properties"` // Array of properties
	CreatedAt       string     `json:"created_at"`
	UpdatedAt       string     `json:"updated_at"`
}

type Property struct {
	Name     string `json:"name"`     // Name of the property
	Type     string `json:"type"`     // Type of the property (e.g., string, integer)
	Required bool   `json:"required"` // Indicate if the property is required
}

// PostgreSQL connection string
const connStr = "postgres://cred-service:cred-service-1@postgres:5432/credential-service?sslmode=disable"

func openDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func insertSchema(schema Schema) (string, error) {
	db, err := openDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	var schemaID string

	// Convert schema.Properties to JSONB format
	schemaJson, err := json.Marshal(schema.Properties)
	if err != nil {
		return "", err
	}

	// Insert the schema into the database
	query := "INSERT INTO schemas (organization_did, schema_name, schema_json) VALUES ($1, $2, $3) RETURNING id"
	err = db.QueryRow(query, schema.OrganizationDID, schema.SchemaName, string(schemaJson)).Scan(&schemaID)
	if err != nil {
		return "", err
	}

	return schemaID, nil
}

// Fetch all schemas
func fetchAllSchemas() ([]Schema, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var schemas []Schema
	query := "SELECT id, organization_did, schema_name, schema_json, created_at, updated_at FROM schemas"

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema Schema
		var schemaJSON []byte

		err := rows.Scan(&schema.ID, &schema.OrganizationDID, &schema.SchemaName, &schemaJSON, &schema.CreatedAt, &schema.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal schemaJSON into the SchemaJSON field
		err = json.Unmarshal(schemaJSON, &schema.Properties)
		if err != nil {
			return nil, err
		}

		schemas = append(schemas, schema)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return schemas, nil
}

// Fetch a schema by its ID
func fetchSchemaByID(id string) (*Schema, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var schema Schema
	var schemaJSON []byte

	query := "SELECT id, organization_did, schema_name, schema_json, created_at, updated_at FROM schemas WHERE id = $1"
	err = db.QueryRow(query, id).Scan(&schema.ID, &schema.OrganizationDID, &schema.SchemaName, &schemaJSON, &schema.CreatedAt, &schema.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Unmarshal the schema_json field
	err = json.Unmarshal(schemaJSON, &schema.Properties)
	if err != nil {
		return nil, err
	}

	return &schema, nil
}

// Update a schema by its ID
func updateSchemaByID(id string, schema Schema) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	query := "UPDATE schemas SET schema_name = $1, schema_json = $2, updated_at = NOW() WHERE id = $3"
	_, err = db.Exec(query, schema.SchemaName, schema.Properties, id)
	if err != nil {
		return err
	}

	return nil
}

// Delete a schema by its ID
func deleteSchemaByID(id string) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	query := "DELETE FROM schemas WHERE id = $1"
	_, err = db.Exec(query, id)
	if err != nil {
		return err
	}

	return nil
}
