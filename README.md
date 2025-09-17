# RunBox

RunBox is a simple **FaaS (Functions as a Service)** platform written in Go.  
It allows you to create, edit, and execute custom JavaScript functions from a web interface.  
Internally, RunBox uses:

- [**SQLite3**](https://github.com/mattn/go-sqlite3) for persistent storage  
- [**Otto**](https://github.com/robertkrimen/otto) as the JavaScript interpreter  

---

## Features
- Write and run JavaScript functions in the browser  
- Persistent storage with SQLite3  
- Lightweight and fast (built with Go)  
- Simple web UI for creating and editing functions  
- Supports HTTP methods (GET, POST, PUT, DELETE, etc.)  

---

## Installation

Clone the repository:

```bash
git clone git@github.com:prodemmi/runbox.git
cd runbox
```

## Install dependencies:
```bash
go mod tidy
```

## Running Locally
Start the server with:
```bash
go run main.go
```
Open your browser at:
http://localhost:8080


## Docker
You can also run RunBox in Docker:
```bash
docker build -t runbox .
docker run -p 8080:8080 runbox
```
Then open: http://localhost:8080

## Example Function
Example of a simple function you can define in the UI:
```javascript
async function POST(request) {
    var body = request.body || {};
    var url = body.url;

    if (!url) {
        return { status: "error", message: "Missing url in request body" };
    }

    try {
        // Call external API
        var response = await fetch("https://is.gd/create.php?format=simple&url=" + encodeURIComponent(url));
        var shortUrl = await response.text();

        return {
            status: "success",
            original: url,
            short: shortUrl,
            provider: "is.gd",
            timestamp: (new Date()).toISOString()
        };
    } catch (err) {
        return { status: "error", message: "Failed to shorten URL", error: String(err) };
    }
}

```