package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"database/sql"

	"google.golang.org/grpc"

	_ "github.com/go-sql-driver/mysql"
	pb "github.com/michaelc445/proto"
)

var (
	port              = flag.Int("port", 50051, "The server port")
	placePosterQuery  = "insert into fyp_schema.posters (partyId, userId, created,updated,location) values (?,?,NOW(),NOW(),point(?,?))"
	checkPosterQuery  = "select partyId, posterId from fyp_schema.posters where posterId = ?"
	removePosterQuery = "DELETE from fyp_schema.posters where posterId = ? and partyId = ?"
)

type server struct {
	pb.UnimplementedPosterAppServer
	DB *sql.DB
}
type Result struct {
	PosterId int32
	PartyId  int32
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
