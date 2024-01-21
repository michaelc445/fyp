package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"database/sql"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"

	_ "github.com/go-sql-driver/mysql"
	pb "github.com/michaelc445/proto"
)

var (
	port                 = flag.Int("port", 50051, "The server port")
	placePosterQuery     = "insert into fyp_schema.posters (partyId, userId, created,updated,location) values (?,?,NOW(),NOW(),point(?,?))"
	checkPosterQuery     = "select partyId, posterId from fyp_schema.posters where posterId = ?"
	removePosterQuery    = "DELETE from fyp_schema.posters where posterId = ? and partyId = ?"
	registerAccountQuery = "insert into fyp_schema.users (partyId, username, pwhash) values (1,?,?)"
	accountExistsQuery   = "select username, userId from fyp_schema.users where username = ?"
	addUserinfoQuery     = "insert into fyp_schema.userinfo (userID, firstName, lastName,location) values (?,?,?,null)"
)

type server struct {
	pb.UnimplementedPosterAppServer
	DB *sql.DB
}
type Result struct {
	PosterId int32
	PartyId  int32
}

type Account struct {
	Username  string
	UserId    int
	PartyId   int
	Pwhash    string
	PartyName string
}

func (s *server) PlacePoster(ctx context.Context, in *pb.PlacementRequest) (*pb.PlacementResponse, error) {
	if in.Location == nil {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("Location of poster not set")
	}
	if in.UserId == 0 {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("userId not set")
	}
	if in.PartyId == 0 {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("posterId not set")
	}
	res, err := s.DB.Exec(placePosterQuery, in.PartyId, in.UserId, in.Location.Lat, in.Location.Lng)
	if err != nil {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to insert poster to database: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to get posterId from query: %v", err)
	}
	return &pb.PlacementResponse{Code: pb.ResponseCode_OK, PosterId: int32(id)}, nil
}

func (s *server) RemovePoster(ctx context.Context, in *pb.RemovePosterRequest) (*pb.RemovePosterResponse, error) {
	if in.UserId == 0 {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("userId not set")
	}
	if in.PosterId == 0 {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("posterId not set")
	}
	if in.PartyId == 0 {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("partyId not set")
	}
	// check that the poster exists / belongs to correct party
	idResponse, err := s.DB.Query(checkPosterQuery, in.PosterId)
	if err != nil {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query database: %v", err)
	}
	defer idResponse.Close()
	// check that there was a response, if not then the poster does not exist
	if !idResponse.Next() {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("no poster found with id: %v", in.PosterId)
	}
	var result Result
	err = idResponse.Scan(&result.PartyId, &result.PosterId)
	if err != nil {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to scan sql result: %v", err)
	}
	if result.PartyId != in.PartyId {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("poster belongs to a different party. expected party: %v got party: %v", in.PartyId, result.PartyId)
	}
	_, err = s.DB.Exec(removePosterQuery, in.PosterId, in.PartyId)
	if err != nil {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to execute query err: %v", err)
	}
	return &pb.RemovePosterResponse{Code: pb.ResponseCode_OK}, nil
}

func (s *server) RegisterAccount(ctx context.Context, in *pb.RegisterAccountRequest) (*pb.RegisterAccountResponse, error) {
	if in.Username == "" {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("username can't be empty")
	}
	if in.FirstName == "" {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("first name can't be empty")
	}
	if in.LastName == "" {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("last name can't be empty")
	}
	if in.Password == "" {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("password can't be empty")
	}
	// check does the username already exist
	res, err := s.DB.Query(accountExistsQuery, in.Username)
	defer res.Close()
	if err != nil {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query username from database: %v", err)
	}
	if res.Next() {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("username already exists")
	}
	pwhash, err := hash(in.Password)
	if err != nil {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to create password hash: %v", err)
	}
	// add acount to users table
	addAccountRes, err := s.DB.Exec(registerAccountQuery, in.Username, pwhash)
	if err != nil {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to add account to database: %v", err)
	}
	userId, err := addAccountRes.LastInsertId()
	if err != nil {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to read userid from database: %v", err)
	}
	// add account to userinfo table
	_, err = s.DB.Exec(addUserinfoQuery, userId, in.FirstName, in.LastName)
	if err != nil {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to add account to userinfo database: %v", err)
	}
	return &pb.RegisterAccountResponse{Code: pb.ResponseCode_OK}, nil
}

func (s *server) LoginAccount(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	if in.Username == "" {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("username not supplied")
	}
	if in.Password == "" {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("password not supplied")
	}

	res, err := s.DB.Query("select users.userID,users.partyID,users.username,users.pwhash,parties.partyName from fyp_schema.users join fyp_schema.parties on users.partyID = parties.partyID where users.username = ?", in.Username)
	if err != nil {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query database for username %v. err: %v", in.Username, err)
	}
	// not returning error, this means username does not exist in database
	if !res.Next() {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to login")
	}
	var result Account
	err = res.Scan(&result.UserId, &result.PartyId, &result.Username, &result.Pwhash, &result.PartyName)
	if err != nil {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to scan sql result: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(result.Pwhash), []byte(in.Password)); err != nil {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to login")
	}

	// generate JWT here and send back in authkey field
	claims := UserClaims{
		UserID:   result.UserId,
		Username: result.Username,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
		},
	}
	accessToken, err := NewAccessToken(claims)
	if err != nil {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to create access token %v", err)
	}

	return &pb.LoginResponse{AuthKey: accessToken, Code: pb.ResponseCode_OK, Party: result.PartyName, UserId: int32(result.UserId), PartyId: int32(result.PartyId)}, nil
}

func hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/fyp_schema")
	defer db.Close()

	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	pb.RegisterPosterAppServer(s, &server{DB: db})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
