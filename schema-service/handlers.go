package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
)

// Create Schema
func createSchema(w http.ResponseWriter, r *http.Request) {
	var schema Schema
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Insert schema into database
	schemaID, err := insertSchema(schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("Schema created with ID: %s", schemaID)))
}

// Get All Schemas
func getAllSchemas(w http.ResponseWriter, r *http.Request) {
	schemas, err := fetchAllSchemas()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(schemas)
}

// Get Schema by ID
func getSchemaByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	schemaID := vars["id"]

	schema, err := fetchSchemaByID(schemaID)
	if err != nil {
		http.Error(w, "Schema not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(schema)
}

// Update Schema
func updateSchema(w http.ResponseWriter, r *http.Request) {
	var schema Schema
	vars := mux.Vars(r)
	schemaID := vars["id"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = updateSchemaByID(schemaID, schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Schema %s updated successfully", schemaID)))
}

// Delete Schema
func deleteSchema(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	schemaID := vars["id"]

	err := deleteSchemaByID(schemaID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Schema %s deleted successfully", schemaID)))
}
