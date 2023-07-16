package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
)

// Função para fazer log de cada requisição
func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methodColor := color.New(color.FgRed).SprintFunc()
		log.Printf("Método: %s | Caminho: %s | IP: %s", methodColor(r.Method), r.URL.Path, r.RemoteAddr)
		handler.ServeHTTP(w, r)
	})
}

// Função para abrir o navegador padrão
func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("sistema operacional não suportado")
	}

	if err != nil {
		log.Println("Erro ao abrir o navegador:", err)
	}
}


// Iniciar o servidor HTTP
func startServer(directory string, port int) {
	// Criando o FileServer com o diretório raiz
	fs := http.FileServer(http.Dir(directory))

	// Criando o manipulador para lidar com as solicitações HTTP
	http.Handle("/", logRequest(http.StripPrefix("/", fs)))

	// Iniciando o servidor na porta especificada
	address := fmt.Sprintf(":%d", port)
	log.Printf("Servidor iniciado. Acesse http://localhost%s", address)
	go openBrowser(fmt.Sprintf("http://localhost%s", address))
	log.Fatal(http.ListenAndServe(address, nil))
}

// Monitorar as alterações no diretório raiz
func monitorChanges(directory string, port int) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			err := watcher.Add(path)
			if err != nil {
				log.Println(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Remove == fsnotify.Remove ||
				event.Op&fsnotify.Rename == fsnotify.Rename ||
				event.Op&fsnotify.Chmod == fsnotify.Chmod {
				// Esperar um curto intervalo de tempo para garantir que todas as alterações tenham ocorrido
				time.Sleep(100 * time.Millisecond)

				// Reiniciar o servidor
				log.Println("Reiniciando o servidor devido a alterações no diretório raiz...")
				go startServer(directory, port)
			}
		case err := <-watcher.Errors:
			log.Println("Erro ao monitorar o diretório raiz:", err)
		}
	}
}

func main() {

	// Definindo as flags para o diretório raiz e a porta
	directory := flag.String("dir", "", "Diretório raiz para servir os arquivos HTML")
	port := flag.Int("port", 8000, "Porta para iniciar o servidor")
	flag.Parse()
	
	// Obtendo o diretório raiz a ser usado
	rootDir := *directory
	if rootDir == "" {
		// Se o diretório não for especificado, use o diretório atual
		var err error
		rootDir, err = filepath.Abs(".")
		if err != nil {
			log.Fatal(err)
		}
	}

	// Iniciar o servidor
	go startServer(rootDir, *port)

	// Monitorar as alterações no diretório raiz
	// go monitorChanges(rootDir, *port)

	select {}
}