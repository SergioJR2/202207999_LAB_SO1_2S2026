package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// HealthResponse representa la respuesta del endpoint /health
type HealthResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	VM        string `json:"VM"`
	Carnet    string `json:"carnet"`
}

// CallResponse representa la respuesta de los endpoints call-apiN
type CallResponse struct {
	ApiName    string `json:"apiname"`
	Message    string `json:"message"`
	Connection bool   `json:"connection"`
	Carnet     string `json:"carnet"`
}

// PeerInfo guarda la VM y la URL base conocidas para un peer
type PeerInfo struct {
	VM  string
	URL string
}

// Configuración cargada desde variables de entorno
var (
	apiName string              // Ej: "API1"
	apiNum  string              // Ej: "1"
	vmName  string              // Ej: "VM1"
	carnet  string              // Carnet del estudiante
	port    string              // Puerto donde escucha
	peers   map[string]PeerInfo // Ej: {"API2": {VM:"VM1", URL:"http://192.168.122.220:8082"}}
)

// healthHandler responde el estado de esta API
func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "UP",
		Message:   fmt.Sprintf("%s is Ready", apiName),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		VM:        vmName,
		Carnet:    carnet,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// callHandler recibe /api{N}/{carnet}/call-api{M} y consulta el /health de la API destino
func callHandler(w http.ResponseWriter, r *http.Request) {
	// Ruta esperada: /api{N}/{carnet}/call-api{M}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	w.Header().Set("Content-Type", "application/json")

	if len(parts) != 3 || !strings.HasPrefix(parts[2], "call-api") {
		http.NotFound(w, r)
		return
	}

	targetNum := strings.TrimPrefix(parts[2], "call-api")
	targetName := "API" + targetNum

	peer, ok := peers[targetName]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CallResponse{
			ApiName:    targetName,
			Message:    fmt.Sprintf("ERROR: %s is not configured as a peer", targetName),
			Connection: false,
			Carnet:     carnet,
		})
		return
	}

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(peer.URL + "/health")

	if err != nil {
		json.NewEncoder(w).Encode(CallResponse{
			ApiName:    targetName,
			Message:    fmt.Sprintf("ERROR: The %s located on the %s is not working", targetName, peer.VM),
			Connection: false,
			Carnet:     carnet,
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var h HealthResponse
	if err := json.Unmarshal(body, &h); err != nil || h.Status != "UP" {
		json.NewEncoder(w).Encode(CallResponse{
			ApiName:    targetName,
			Message:    fmt.Sprintf("ERROR: The %s located on the %s is not working", targetName, peer.VM),
			Connection: false,
			Carnet:     carnet,
		})
		return
	}

	json.NewEncoder(w).Encode(CallResponse{
		ApiName:    targetName,
		Message:    fmt.Sprintf("The %s located on the %s is working", targetName, h.VM),
		Connection: true,
		Carnet:     carnet,
	})
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// parsePeers convierte "API2=VM1|http://ip:puerto,API3=VM2|http://ip:puerto" en un map
func parsePeers(s string) map[string]PeerInfo {
	m := map[string]PeerInfo{}
	if s == "" {
		return m
	}
	for _, pair := range strings.Split(s, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		name := strings.TrimSpace(kv[0])
		vmAndURL := strings.SplitN(kv[1], "|", 2)
		if len(vmAndURL) != 2 {
			continue
		}
		m[name] = PeerInfo{
			VM:  strings.TrimSpace(vmAndURL[0]),
			URL: strings.TrimSpace(vmAndURL[1]),
		}
	}
	return m
}

func main() {
	apiName = envOrDefault("APINAME", "API1")
	apiNum = envOrDefault("APINUM", "1")
	vmName = envOrDefault("VM", "VM1")
	carnet = envOrDefault("CARNET", "000000000")
	port = envOrDefault("PORT", "8080")
	peers = parsePeers(os.Getenv("PEERS"))

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc(fmt.Sprintf("/api%s/%s/", apiNum, carnet), callHandler)

	log.Printf("%s iniciando en puerto %s (VM=%s, carnet=%s, peers=%v)\n", apiName, port, vmName, carnet, peers)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}