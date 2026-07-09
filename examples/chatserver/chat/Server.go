package chat

import "github.com/cookiengineer/gows"
import "example/chat/structs"
import "fmt"
import "log"
import "io/fs"
import "net/http"
import "strings"
import "sync"

type Server struct {
	Rooms   map[string]*structs.Room
	Server  *gows.Server
	Handler http.Handler
	mutex   sync.Mutex
}

func NewServer() *Server {

	public_fs, err := fs.Sub(Filesystem, "public")

	if err == nil {

		http_handler := http.FileServer(http.FS(public_fs))

		return &Server{
			Rooms:   make(map[string]*structs.Room),
			Server:  &gows.Server{},
			Handler: http_handler,
			mutex:   sync.Mutex{},
		}

	} else {
		return nil
	}

}

func (server *Server) GetRoom(name string) *structs.Room {

	name = strings.ToLower(name)

	server.mutex.Lock()
	defer server.mutex.Unlock()

	room, ok := server.Rooms[name]

	if ok == false {

		tmp := structs.NewRoom(name)
		server.Rooms[name] = tmp
		room = server.Rooms[name]

	}

	return room

}

func (server *Server) RemoveRoom(name string) {

	name = strings.ToLower(name)

	server.mutex.Lock()
	defer server.mutex.Unlock()

	delete(server.Rooms, name)

}

func (server *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {

	path := request.URL.Path

	if strings.HasPrefix(path, "/api/chat/") {

		room_name := strings.ToLower(strings.TrimPrefix(path, "/api/chat/"))

		if room_name == "" {
			room_name = "welcome"
		}

		if strings.HasPrefix(room_name, "#") {
			room_name = strings.TrimPrefix(room_name, "#")
		}

		server.Upgrade(response, request, room_name)

	} else {

		server.Handler.ServeHTTP(response, request)

	}

}

func (server *Server) Upgrade(response http.ResponseWriter, request *http.Request, room_name string) {

	websocket, err0 := server.Server.Upgrade(response, request)

	if err0 == nil {

		room := server.GetRoom(room_name)
		room.Join(websocket)

		websocket.OnMessage = func(frame []byte) {

			message := structs.ParseMessage(frame)

			if message != nil {

				log.Printf("server: #%s -> %s: %s", room_name, message.User, message.Text)
				room.Broadcast(websocket, *message)

			}

		}

		websocket.OnClose = func() {

			log.Printf("server: #%s -> client disconnected", room_name)

			room.Leave(websocket)

			amount := room.AmountOfClients()

			if amount == 0 {
				server.RemoveRoom(room_name)
			}

		}

		go websocket.Init()

		log.Printf("server: #%s -> client connected", room_name)

	} else {

		error_message := fmt.Sprintf("server: Upgrade error for room %s: %s", room_name, err0.Error())
		http.Error(response, error_message, http.StatusInternalServerError)
		log.Println(error_message)

	}

}
