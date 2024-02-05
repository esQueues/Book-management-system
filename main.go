package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"html/template"
	"log"
	"net/http"
)

var store = sessions.NewCookieStore([]byte("your-secret-key"))

var client *mongo.Client

type User struct {
	Fullname string `json:"fullname"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Book struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Genre string `json:"genre"`
	PublicationYear int32     `json:"publicationYear"`
	ISBN string `json:"isbn"`
}

func init() {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	var err error
	client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		panic(err)
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fullname := r.FormValue("fullname")
		username := r.FormValue("username")
		password := r.FormValue("password")
		user := User{
			Fullname: fullname,
			Username: username,
			Password: password,
		}

		collection := client.Database("project").Collection("users")
		_, err = collection.InsertOne(context.Background(), user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("User registered successfully"))
		return
	}

	http.ServeFile(w, r, "frontend/register.html")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		collection := client.Database("project").Collection("users")
		filter := bson.M{"username": username}
		var storedUser User
		err = collection.FindOne(context.Background(), filter).Decode(&storedUser)
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if storedUser.Password != password {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		session, err := store.Get(r, "session-name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		session.Values["username"] = storedUser.Username
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Login successful. Redirect to profile..."))
		return
	}

	http.ServeFile(w, r, "frontend/login.html")
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	username, ok := session.Values["username"].(string)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	collection := client.Database("project").Collection("users")
	filter := bson.M{"username": username}
	var storedUser User
	err = collection.FindOne(context.Background(), filter).Decode(&storedUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("frontend/profile.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, storedUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	username, ok := session.Values["username"].(string)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	collection := client.Database("project").Collection("users")
	filter := bson.M{"username": username}
	_, err = collection.DeleteOne(context.Background(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Options.MaxAge = -1
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
func editHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	username, ok := session.Values["username"].(string)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	collection := client.Database("project").Collection("users")
	filter := bson.M{"username": username}
	var storedUser User
	err = collection.FindOne(context.Background(), filter).Decode(&storedUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("frontend/edit.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, storedUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	username, ok := session.Values["username"].(string)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	err = r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newFullName := r.FormValue("fullname")
	newUserName := r.FormValue("username")
	newPassword := r.FormValue("password")

	collection := client.Database("project").Collection("users")
	filter := bson.M{"username": username}

	update := bson.M{
		"$set": bson.M{
			"fullname": newFullName,
			"username": newUserName,
			"password": newPassword,
		},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)

}
func filterBooksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		filterValue := r.URL.Query().Get("filter")
		sortValue := r.URL.Query().Get("sort")

		filter := bson.M{"title": bson.M{"$regex": filterValue, "$options": "i"}}

		books, err := getFilteredBooks(filter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if sortValue != "" {
			books = sortBooks(books, sortValue)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(books)
	}
}

func getFilteredBooks(filter bson.M) ([]Book, error) {
	collection := client.Database("project").Collection("books")

	indexModel := mongo.IndexModel{
		Keys: bson.M{"title": 1},
	}

	indexOptions := options.CreateIndexes().SetMaxTime(2 * time.Second) 

	_, err := collection.Indexes().CreateOne(context.Background(), indexModel, indexOptions)
	if err != nil {
		return nil, err
	}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var books []Book
	for cursor.Next(context.Background()) {
		var book Book
		err := cursor.Decode(&book)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}

	return books, nil
}

func sortBooks(books []Book, sortBy string) []Book {
	switch sortBy {
	case "title":
		sort.Slice(books, func(i, j int) bool {
			return books[i].Title < books[j].Title
		})
	case "author":
		sort.Slice(books, func(i, j int) bool {
			return books[i].Author < books[j].Author
		})
	case "genre":
		sort.Slice(books, func(i, j int) bool {
			return books[i].Genre < books[j].Genre
		})
	case "publicationYear":
		sort.Slice(books, func(i, j int) bool {
			return books[i].PublicationYear < books[j].PublicationYear
		})
	default:
		fmt.Println("Invalid sortBy parameter")
	}

	return books
}


func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received request for /")
		http.ServeFile(w, r, "frontend/index.html")
	})
	r.HandleFunc("/register", registerHandler).Methods("POST", "GET")
	r.HandleFunc("/login", loginHandler).Methods("POST", "GET")
	r.HandleFunc("/profile", profileHandler).Methods("GET")
	r.HandleFunc("/delete", deleteHandler).Methods("POST")
	r.HandleFunc("/edit", editHandler).Methods("GET")
	r.HandleFunc("/update", updateHandler).Methods("POST")

	fmt.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", r)

	if err != nil {
		log.Fatal("Error starting the server:", err)
		return
	}
}
