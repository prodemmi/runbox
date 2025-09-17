package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/robertkrimen/otto"
)

type Function struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Path        string `json:"path" db:"path"`
	Code        string `json:"code" db:"code"`
	Description string `json:"description" db:"description"`
}

type App struct {
	db *sql.DB
}

func MethodOverride() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" {
			if override := c.PostForm("_method"); override != "" {
				c.Request.Method = override
			}
		}
		c.Next()
	}
}

func main() {
	app := &App{}
	app.initDB()
	defer app.db.Close()

	r := gin.Default()

	r.Use(MethodOverride())

	r.LoadHTMLGlob("templates/*")

	r.Static("/static", "./static")

	r.GET("/", app.homePage)
	r.GET("/functions/create", app.newFunctionPage)
	r.GET("/functions/:id/edit", app.editFunctionPage)
	r.POST("/api/functions", app.createFunction)
	r.PUT("/api/functions/:id", app.updateFunction)
	r.DELETE("/api/functions/:id", app.deleteFunction)

	r.GET("/api/execute/*path", app.executeFunction)
	r.POST("/api/execute/*path", app.executeFunction)
	r.PUT("/api/execute/*path", app.executeFunction)
	r.PATCH("/api/execute/*path", app.executeFunction)
	r.DELETE("/api/execute/*path", app.executeFunction)
	r.HEAD("/api/execute/*path", app.executeFunction)
	r.OPTIONS("/api/execute/*path", app.executeFunction)

	log.Println("RunBox server starting on :8080")
	r.Run(":8080")
}

func (app *App) initDB() {
	var err error
	app.db, err = sql.Open("sqlite3", "./runbox.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS functions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		path TEXT NOT NULL UNIQUE,
		code TEXT NOT NULL,
		description TEXT
	);`

	_, err = app.db.Exec(createTable)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

func (app *App) homePage(c *gin.Context) {
	functions, err := app.getAllFunctions()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
		return
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":     "RunBox - Function Executor",
		"functions": functions,
	})
}

func (app *App) newFunctionPage(c *gin.Context) {
	c.HTML(http.StatusOK, "function_form.html", gin.H{
		"title":    "Create New Function",
		"function": Function{},
		"action":   "/api/functions",
		"method":   "POST",
	})
}

func (app *App) createFunction(c *gin.Context) {
	var function Function

	function.Name = c.PostForm("name")
	function.Path = c.PostForm("path")
	function.Code = c.PostForm("code")
	function.Description = c.PostForm("description")

	if function.Name == "" || function.Path == "" || function.Code == "" {
		c.HTML(http.StatusBadRequest, "function_form.html", gin.H{
			"title":    "Create New Function",
			"function": function,
			"action":   "/api/functions",
			"method":   "POST",
			"error":    "Name, Path, and Code are required fields",
		})
		return
	}

	if !strings.HasPrefix(function.Path, "/") {
		function.Path = "/" + function.Path
	}

	query := `INSERT INTO functions (name, path, code, description) VALUES (?, ?, ?, ?)`
	result, err := app.db.Exec(query, function.Name, function.Path, function.Code, function.Description)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "function_form.html", gin.H{
			"title":    "Create New Function",
			"function": function,
			"action":   "/api/functions",
			"method":   "POST",
			"error":    "Failed to create function: " + err.Error(),
		})
		return
	}

	id, _ := result.LastInsertId()
	function.ID = int(id)

	c.Redirect(http.StatusFound, "/")
}

func (app *App) editFunctionPage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid function ID"})
		return
	}

	function, err := app.getFunctionByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Function not found"})
		return
	}

	c.HTML(http.StatusOK, "function_form.html", gin.H{
		"title":    "Edit Function",
		"function": function,
		"action":   "/api/functions/" + strconv.Itoa(id),
		"method":   "PUT",
	})
}

func (app *App) updateFunction(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid function ID"})
		return
	}

	var function Function
	function.ID = id
	function.Name = c.PostForm("name")
	function.Path = c.PostForm("path")
	function.Code = c.PostForm("code")
	function.Description = c.PostForm("description")

	if function.Name == "" || function.Path == "" || function.Code == "" {
		c.HTML(http.StatusBadRequest, "function_form.html", gin.H{
			"title":    "Edit Function",
			"function": function,
			"action":   "/api/functions/" + strconv.Itoa(id) + "/",
			"method":   "PUT",
			"error":    "Name, Path, and Code are required fields",
		})
		return
	}

	if !strings.HasPrefix(function.Path, "/") {
		function.Path = "/" + function.Path
	}

	query := `UPDATE functions SET name = ?, path = ?, code = ?, description = ? WHERE id = ?`
	_, err = app.db.Exec(query, function.Name, function.Path, function.Code, function.Description, id)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "function_form.html", gin.H{
			"title":    "Edit Function",
			"function": function,
			"action":   "/api/functions/" + strconv.Itoa(id),
			"method":   "PUT",
			"error":    "Failed to update function: " + err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, "/")
}

func (app *App) deleteFunction(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid function ID"})
		return
	}

	query := `DELETE FROM functions WHERE id = ?`
	_, err = app.db.Exec(query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete function"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Function deleted successfully"})
}

func (app *App) executeFunction(c *gin.Context) {
	path := c.Param("path")

	function, err := app.getFunctionByPath(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Function not found"})
		return
	}

	result, err := app.executeJavaScript(function.Code, c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "Function execution failed",
			"details":  err.Error(),
			"function": function.Name,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (app *App) getAllFunctions() ([]Function, error) {
	query := `SELECT id, name, path, code, description FROM functions ORDER BY name`
	rows, err := app.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []Function
	for rows.Next() {
		var f Function
		err := rows.Scan(&f.ID, &f.Name, &f.Path, &f.Code, &f.Description)
		if err != nil {
			return nil, err
		}
		functions = append(functions, f)
	}

	return functions, nil
}

func (app *App) getFunctionByID(id int) (*Function, error) {
	query := `SELECT id, name, path, code, description FROM functions WHERE id = ?`
	row := app.db.QueryRow(query, id)

	var f Function
	err := row.Scan(&f.ID, &f.Name, &f.Path, &f.Code, &f.Description)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

func (app *App) getFunctionByPath(path string) (*Function, error) {
	query := `SELECT id, name, path, code, description FROM functions WHERE path = ?`
	row := app.db.QueryRow(query, path)

	var f Function
	err := row.Scan(&f.ID, &f.Name, &f.Path, &f.Code, &f.Description)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

func (app *App) executeJavaScript(code string, c *gin.Context) (interface{}, error) {
	vm := otto.New()

	requestData := map[string]interface{}{
		"method":  c.Request.Method,
		"path":    c.Request.URL.Path,
		"query":   map[string]interface{}{},
		"body":    map[string]interface{}{},
		"headers": map[string]string{},
	}

	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			requestData["query"].(map[string]interface{})[key] = values[0]
		}
	}

	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err == nil && len(bodyBytes) > 0 {
			var bodyData map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &bodyData); err == nil {
				requestData["body"] = bodyData
			} else {

				requestData["body"] = map[string]interface{}{"raw": string(bodyBytes)}
			}
		}
	}

	if c.Request.Method == "POST" || c.Request.Method == "PUT" {
		if err := c.Request.ParseForm(); err == nil {
			for key, values := range c.Request.PostForm {
				if len(values) > 0 {
					if requestData["body"] == nil {
						requestData["body"] = map[string]interface{}{}
					}
					if bodyMap, ok := requestData["body"].(map[string]interface{}); ok {
						bodyMap[key] = values[0]
					}
				}
			}
		}
	}

	for key, values := range c.Request.Header {
		if len(values) > 0 {
			requestData["headers"].(map[string]string)[key] = values[0]
		}
	}

	vm.Set("request", requestData)

	vm.Set("console", map[string]interface{}{
		"log": func(args ...interface{}) {
			log.Println("JS Console:")
		},
	})

	_, err := vm.Run(code)
	if err != nil {
		return nil, fmt.Errorf("JavaScript execution error: %v", err)
	}

	methodName := strings.ToUpper(c.Request.Method)

	if fn, err := vm.Get(methodName); err == nil && fn.IsFunction() {

		result, err := fn.Call(otto.UndefinedValue(), requestData)
		if err != nil {
			return nil, fmt.Errorf("error calling %s handler: %v", methodName, err)
		}

		if result.IsDefined() {
			goValue, err := result.Export()
			if err != nil {
				return nil, fmt.Errorf("failed to export result: %v", err)
			}
			return goValue, nil
		}
	}

	if defaultFn, err := vm.Get("default"); err == nil && defaultFn.IsFunction() {
		result, err := defaultFn.Call(otto.UndefinedValue(), requestData)
		if err != nil {
			return nil, fmt.Errorf("error calling default handler: %v", err)
		}

		if result.IsDefined() {
			goValue, err := result.Export()
			if err != nil {
				return nil, fmt.Errorf("failed to export result: %v", err)
			}
			return goValue, nil
		}
	}

	return map[string]interface{}{
		"error":             fmt.Sprintf("No handler found for method %s. Please define a %s function or a default function.", methodName, methodName),
		"availableHandlers": []string{"GET", "POST", "PUT", "PATCH", "DELETE", "default"},
		"method":            methodName,
	}, nil
}
