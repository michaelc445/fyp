package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"net"
	"time"

	"database/sql"
	"github.com/golang-jwt/jwt"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/michaelc445/fyp/tokenService"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"

	_ "github.com/go-sql-driver/mysql"
	pb "github.com/michaelc445/proto"
)

var (
	removePosterMaxDistance = 30
	port                    = flag.Int("port", 50051, "The server port")
	placePosterQuery        = "insert into fyp_schema.posters (partyId, userId, created,updated,location) values (?,?,NOW(),NOW(),point(?,?))"
	checkPosterQuery        = "select partyId, posterId from fyp_schema.posters where posterId = ?"
	outstandingPosterQuery  = `	select unix_timestamp(l2.created), l2.posterId, l2.userId, l4.username,l3.firstName, l3.lastName
								from fyp_schema.elections as l1
								join fyp_schema.posters as l2 on l1.partyId = l2.partyID
								join fyp_schema.userinfo as l3 on l2.userID = l3.userID
								join fyp_schema.users as l4 on l2.userId = l4.userId
								where l1.partyId = ? and l2.removed is null and l2.created > l1.startDate;`
	removePosterQuery    = "update fyp_schema.posters set removed = now(), updated = now(), removedBy = ? where posterID = ? and partyID = ?;"
	registerAccountQuery = "insert into fyp_schema.users (partyId, username, pwhash) values (1,?,?)"
	accountExistsQuery   = "select username, userId from fyp_schema.users where username = ?"
	addUserinfoQuery     = "insert into fyp_schema.userinfo (userID, firstName, lastName,location) values (?,?,?,null)"
	posterDistanceQuery  = "select posterID, ST_Distance_Sphere(location, point(?,?)) as distance from fyp_schema.posters where partyID = ? and removed is null having distance < ? order by distance asc limit 1;"
	userInfoQuery        = "select users.userID,users.partyID,users.username,users.pwhash,parties.partyName from fyp_schema.users join fyp_schema.parties on users.partyID = parties.partyID where users.username = ?"
	joinRequestQuery     = "select t1.userid, t2.firstName, t2.lastname from fyp_schema.joinRequests as t1 join fyp_schema.userinfo as t2 on t1.userID = t2.userID where t1.partyId = ? and t1.reviewed = false"
)

type server struct {
	pb.UnimplementedPosterAppServer
	DB *sql.DB
}
type Account struct {
	Username  string
	UserId    int
	PartyId   int
	Pwhash    string
	PartyName string
}
type Poster struct {
	posterId int32
	distance float64
}
type PosterUpdate struct {
	PosterId int32
	PartyId  int32
	UserID   int32

	Removed  timestamp.Timestamp
	location pb.Location
}
type OutstandingPoster struct {
	created   int64
	posterId  int32
	userID    int32
	username  string
	firstName string
	lastName  string
}

func hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
func verifyClaims(claims *tokenService.UserClaims, userId int32, partyId int32) bool {
	if claims.UserID != userId || claims.PartyId != partyId {
		return false
	}
	return true
}

// OutstandingPosters retrieves information about poster which have not yet been removed and the date by which they must be removed.
func (s *server) OutstandingPosters(ctx context.Context, in *pb.PosterTimeRequest) (*pb.PosterTimeResponse, error) {
	if in.GetAuthKey() == "" {
		return &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authkey not set")
	}
	if in.GetUserId() == 0 {
		return &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("userid not set")
	}
	if in.GetPartyId() == 0 {
		return &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("partyId not set")
	}
	userClaims := tokenService.ParseAccessToken(in.GetAuthKey())
	if userClaims == nil || userClaims.Valid() != nil {
		return &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey is invalid. please login again")
	}
	if !verifyClaims(userClaims, in.GetUserId(), in.GetPartyId()) {
		return &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey does not match supplied id's. Please login again")
	}
	rows, err := s.DB.Query(outstandingPosterQuery, in.GetPartyId())
	if err != nil {
		return &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query outsatanding posters: %v", err)
	}

	var posters []*pb.PosterUser
	for rows.Next() {
		var poster OutstandingPoster
		err = rows.Scan(&poster.created, &poster.posterId, &poster.userID, &poster.username, &poster.firstName, &poster.lastName)
		if err != nil {
			return &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to read from from sql result: %v", err)
		}
		posters = append(posters, &pb.PosterUser{
			Poster:    &pb.Poster{Posterid: poster.posterId, PlacedBy: poster.userID},
			Created:   timestamppb.New(time.Unix(poster.created, 0)),
			Username:  poster.username,
			FirstName: poster.firstName,
			LastName:  poster.lastName,
		})
	}
	rows.Close()
	rows, err = s.DB.Query("select unix_timestamp(endDate) from fyp_schema.elections where partyId = ?", in.GetPartyId())
	if err != nil {
		return &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query election end date: %v", err)
	}
	if !rows.Next() {
		return &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("party admin must create an election first")
	}
	var electionDate int64
	err = rows.Scan(&electionDate)
	if err != nil {
		return &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to read election date from query: %v", err)
	}
	rows.Close()
	return &pb.PosterTimeResponse{Code: pb.ResponseCode_OK, Posters: posters, RemovalDate: timestamppb.New(time.Unix(electionDate, 0))}, nil
}

// NewElection updates the current election dates for a party
func (s *server) NewElection(ctx context.Context, in *pb.CreateElectionRequest) (*pb.CreateElectionResponse, error) {
	if in.GetAuthKey() == "" {
		return &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authkey not set")
	}
	if in.GetUserId() == 0 {
		return &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("userid not set")
	}
	if in.GetPartyId() == 0 {
		return &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("partyId not set")
	}
	userClaims := tokenService.ParseAccessToken(in.GetAuthKey())
	if userClaims == nil || userClaims.Valid() != nil {
		return &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey is invalid. please login again")
	}
	if !verifyClaims(userClaims, in.GetUserId(), in.GetPartyId()) {
		return &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey does not match supplied id's. Please login again")
	}
	// check the user is admin
	rows, err := s.DB.Query("select * from fyp_schema.parties where partyID = ? and admin = ?", in.GetPartyId(), in.GetUserId())

	if err != nil {
		return &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to check permissions: %v", err)
	}
	if !rows.Next() {
		return &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("only party admin can create a new election")
	}
	// check that election is in the future
	if in.GetElectionDate().AsTime().UnixMilli() <= time.Now().UnixMilli() {
		return &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("election must be in the future")
	}
	// check that date of election is after the start date of election
	if in.GetStartDate().GetSeconds() >= in.GetElectionDate().GetSeconds() {
		return &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("election date must come after the start date")
	}

	// add election to database, replacing the old election if it exists
	_, err = s.DB.Exec("replace into fyp_schema.elections (partyId, startDate, endDate) values (?,from_unixtime(?),from_unixtime(?))", in.GetPartyId(), in.GetStartDate().AsTime().Unix(), in.GetElectionDate().AsTime().Unix())

	if err != nil {
		return &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to update election %v", err)
	}

	return &pb.CreateElectionResponse{Code: pb.ResponseCode_OK}, nil
}

// RetrieveProfileStats retrieves information about a given user.
func (s *server) RetrieveProfileStats(ctx context.Context, in *pb.ProfileRequest) (*pb.ProfileResponse, error) {

	if in.GetPartyId() == 0 {
		return &pb.ProfileResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("partyId not set")
	}
	if in.GetUserId() == 0 {
		return &pb.ProfileResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("userId not set")
	}
	if in.GetAuthKey() == "" {
		return &pb.ProfileResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey not set")
	}
	userClaims := tokenService.ParseAccessToken(in.GetAuthKey())
	if userClaims == nil || userClaims.Valid() != nil {
		return &pb.ProfileResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey is invalid. please login again")
	}
	if !verifyClaims(userClaims, in.GetUserId(), in.GetPartyId()) {
		return &pb.ProfileResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey does not match supplied id's. Please login again")
	}

	// get number of posters placed and removed by user
	rows, err := s.DB.Query(
		`		select l1.placed, l2.removed, l3.partyName 
					from (select count(posterId) as placed from fyp_schema.posters where userID = ?) as l1 
				    join (select count(posterId) as removed from fyp_schema.posters where removedBy = ?) as l2 
				    join (select partyName from fyp_schema.parties where partyId = ?) as l3;`, in.GetUserId(), in.GetUserId(), in.GetPartyId())
	if err != nil {
		return &pb.ProfileResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query profile statistics: %v", err)
	}

	if !rows.Next() {
		return &pb.ProfileResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query database")
	}
	placed, removed, partyName := 0, 0, ""
	err = rows.Scan(&placed, &removed, &partyName)
	if err != nil {
		return &pb.ProfileResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to read sql response: %v", err)
	}

	return &pb.ProfileResponse{Code: pb.ResponseCode_OK, PlacedPosters: int32(placed), RemovedPosters: int32(removed), PartyName: partyName}, nil
}

// ApproveMembers will approve or deny members from joining a party.
func (s *server) ApproveMembers(ctx context.Context, in *pb.ApproveMemberRequest) (*pb.ApproveMemberResponse, error) {
	if in.GetPartyId() == 0 {
		return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("partyId not set")
	}
	if in.GetUserId() == 0 {
		return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("userID not set")
	}
	if in.GetAuthKey() == "" {
		return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authkey not set")
	}
	userClaims := tokenService.ParseAccessToken(in.GetAuthKey())
	if userClaims == nil || userClaims.Valid() != nil {
		return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey is invalid. please login again")
	}
	if !verifyClaims(userClaims, in.GetUserId(), in.GetPartyId()) {
		return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey does not match supplied id's. Please login again")
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to start transaction %v", err)
	}

	//check that the user is admin of the party
	rows, err := s.DB.Query("select * from fyp_schema.parties where partyId = ? and admin = ?", in.GetPartyId(), in.GetUserId())
	if err != nil {
		return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query party table: %v", err)
	}
	if !rows.Next() {
		return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("to approve members you must be admin of the party")
	}
	_ = rows.Close()
	if approvedMembers := in.GetApprovedMembers(); approvedMembers != nil {
		for _, member := range approvedMembers {
			//check there is a request to join from this member
			rows, err := tx.Query("select id from fyp_schema.joinRequests where userId = ? and partyid = ? and reviewed = false",
				member.GetUserId(),
				in.GetPartyId(),
			)
			if err != nil {
				_ = tx.Rollback()
				return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query join requests: %v", err)
			}
			// no join request from this user
			if !rows.Next() {
				_ = tx.Rollback()
				return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("no join request from this user: %s %s", member.GetFirstName(), member.GetLastName())
			}
			err = rows.Close()
			if err != nil {
				return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to close rows: %v", err)
			}
			// update users party
			_, err = tx.Exec("update fyp_schema.users set partyId = ? where userId = ?", in.GetPartyId(), member.GetUserId())
			if err != nil {
				_ = tx.Rollback()
				return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to update user: %v", err)
			}
			// update users posters to belong to new party
			_, err = tx.Exec("update fyp_schema.posters set partyId = ? where userId = ? and posterId > 0", in.GetPartyId(), member.GetUserId())
			if err != nil {
				_ = tx.Rollback()
				return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to update users posters: %v", err)
			}
			// set request = reviewed
			_, err = tx.Exec("update fyp_schema.joinRequests set reviewed = true where userId = ? and partyId = ?", member.GetUserId(), in.GetPartyId())
			if err != nil {
				_ = tx.Rollback()
				return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to update join request: %v", err)
			}
		}
	}

	if deniedMembers := in.GetDeniedMembers(); deniedMembers != nil {
		for _, member := range deniedMembers {
			// set members join request to reviewed
			_, err = tx.Exec("update fyp_schema.joinRequests set reviewed = true where userId = ? and partyId = ?", member.GetUserId(), in.GetPartyId())
			if err != nil {
				_ = tx.Rollback()
				return &pb.ApproveMemberResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to update join request: %v", err)
			}
		}
	}
	_ = tx.Commit()
	return &pb.ApproveMemberResponse{Code: pb.ResponseCode_OK}, nil
}

// RetrieveJoinRequests will return a list of outstanding join requests for a party.
func (s *server) RetrieveJoinRequests(ctx context.Context, in *pb.RetrieveJoinRequest) (*pb.RetrieveJoinResponse, error) {

	if in.GetPartyId() == 0 {
		return &pb.RetrieveJoinResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("partyId not set")
	}
	if in.GetUserId() == 0 {
		return &pb.RetrieveJoinResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("userID not set")
	}
	if in.GetAuthKey() == "" {
		return &pb.RetrieveJoinResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authkey not set")
	}
	userClaims := tokenService.ParseAccessToken(in.GetAuthKey())
	if userClaims == nil || userClaims.Valid() != nil {
		return &pb.RetrieveJoinResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey is invalid. please login again")
	}
	if !verifyClaims(userClaims, in.GetUserId(), in.GetPartyId()) {
		return &pb.RetrieveJoinResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey does not match supplied id's. Please login again")
	}
	//check that the user is admin of the party
	rows, err := s.DB.Query("select * from fyp_schema.parties where partyId = ? and admin = ?", in.GetPartyId(), in.GetUserId())
	if err != nil {
		return &pb.RetrieveJoinResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query party table: %v", err)
	}
	if !rows.Next() {
		return &pb.RetrieveJoinResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("to retrieve join requests you must be admin of the party")
	}
	_ = rows.Close()
	//retrieve unreviewed join requests
	rows, err = s.DB.Query(joinRequestQuery, in.GetPartyId())
	var memberList []*pb.Member
	for rows.Next() {
		var member pb.Member
		err := rows.Scan(&member.UserId, &member.FirstName, &member.LastName)
		if err != nil {
			return &pb.RetrieveJoinResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("error while reading from database %v", err)
		}
		memberList = append(memberList, &member)
	}
	_ = rows.Close()
	return &pb.RetrieveJoinResponse{Code: pb.ResponseCode_OK, Members: memberList}, nil
}

// JoinParty will add a user to a specific party.
func (s *server) JoinParty(ctx context.Context, in *pb.JoinPartyRequest) (*pb.JoinPartyResponse, error) {

	if in.UserId == 0 {
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("userID not set")
	}
	if in.GetPartyId() == 0 {
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("partyID not set")
	}
	if in.GetAuthKey() == "" {
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey not set")
	}
	userClaims := tokenService.ParseAccessToken(in.GetAuthKey())
	if userClaims == nil || userClaims.Valid() != nil || userClaims.UserID != in.UserId {
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey is invalid. please login again")
	}

	// check that party exists
	rows, err := s.DB.Query("select * from fyp_schema.parties where partyId = ?", in.GetPartyId())
	if err != nil {
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query party table %v", err)
	}
	if !rows.Next() {
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("party does not exist")
	}
	_ = rows.Close()
	// check user is not already a member of a party
	rows, err = s.DB.Query("select userID, partyID from fyp_schema.users where userID = ? and partyId > 1", in.GetUserId())
	if err != nil {
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query party table: %v", err)
	}
	// should have no rows in response
	if rows.Next() {
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("can not join party while you are a member of a party")
	}
	_ = rows.Close()
	//check that user has not already requested to join party
	rows, err = s.DB.Query("select * from fyp_schema.joinRequests where userID = ? and partyID = ? and reviewed = false", in.GetUserId(), in.GetPartyId())
	if err != nil {
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query party table: %v", err)
	}
	// should have no rows in response
	if rows.Next() {
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("request still pending")
	}
	_ = rows.Close()

	// set all active join requests for this user to reviewed
	// i.e user can only have 1 pending request at a time
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to start transaction: %v", err)
	}
	_, err = tx.Exec("update fyp_schema.joinRequests set reviewed = true where userID = ? and id > 0 and reviewed = false", in.GetUserId())
	if err != nil {
		_ = tx.Rollback()
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to reset join requests: %v", err)
	}
	// create update in party request table
	_, err = tx.Exec("insert into fyp_schema.joinRequests (userID,partyID) values(?,?)", in.GetUserId(), in.GetPartyId())
	if err != nil {
		_ = tx.Rollback()
		return &pb.JoinPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to create join request")
	}
	_ = tx.Commit()
	return &pb.JoinPartyResponse{Code: pb.ResponseCode_OK}, nil
}

// RegisterParty will allow a user to create a new party with the user as party admin
func (s *server) RegisterParty(ctx context.Context, in *pb.RegisterPartyRequest) (*pb.RegisterPartyResponse, error) {
	if in.PartyName == "" {
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("party name can not be empty")
	}
	if in.GetUserId() == 0 {
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("userID not set")
	}
	if in.GetAuthKey() == "" {
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("auth key not set")
	}

	// verify authkey
	userClaims := tokenService.ParseAccessToken(in.GetAuthKey())
	if userClaims == nil || userClaims.Valid() != nil || userClaims.UserID != in.UserId {
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey is invalid. please login again")
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to start transaction %v", err)
	}

	// check if user is a member of a party (can't create new party if you are a member of a party)
	res, err := tx.Query("select userID, partyID from fyp_schema.users where userId = ? and partyId > 1", in.GetUserId())
	if err != nil {
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to lookup users party: %v", err)
	}
	// there should be no rows returned
	if res.Next() {
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("can't register new party while you are a member of a party")
	}
	_ = res.Close()
	// check if party exists
	res, err = tx.Query("select * from fyp_schema.parties where partyName = ?", in.GetPartyName())
	if err != nil {
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to search party database: %v", err)
	}
	// there should be no rows returned
	if res.Next() {
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("party already exists")
	}
	_ = res.Close()
	// create new party with user as admin
	rows, err := tx.Exec("insert into fyp_schema.parties (partyName, admin) values (?,?)", in.GetPartyName(), in.GetUserId())
	if err != nil {
		_ = tx.Rollback()
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to create new party %v", err)
	}
	partyId, err := rows.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to create new party %v", err)
	}

	// change users party to the new party
	rows, err = tx.Exec("update fyp_schema.users set partyID = ? where userID = ?", partyId, in.GetUserId())
	if err != nil {
		_ = tx.Rollback()
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed update users party %v", err)
	}
	numRows, err := rows.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to check number of rows affected %v", err)
	}
	if numRows != 1 {
		_ = tx.Rollback()
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to update users party")
	}
	userClaims.PartyId = int32(partyId)
	authKey, err := tokenService.NewAccessToken(*userClaims)
	if err != nil {
		_ = tx.Rollback()
		return &pb.RegisterPartyResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to create new authKey %v", err)
	}
	_ = tx.Commit()
	return &pb.RegisterPartyResponse{Code: pb.ResponseCode_OK, PartyId: int32(partyId), AuthKey: authKey}, nil
}

// PlacePoster will add a poster to the database at a specific location.
func (s *server) PlacePoster(ctx context.Context, in *pb.PlacementRequest) (*pb.PlacementResponse, error) {
	if in.GetLocation() == nil {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("location of poster not set")
	}
	if in.GetUserId() == 0 {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("userId not set")
	}
	if in.GetPartyId() == 0 {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("posterId not set")
	}
	if in.GetAuthKey() == "" {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey not set")
	}

	// verify authkey
	userClaims := tokenService.ParseAccessToken(in.GetAuthKey())
	if userClaims == nil || userClaims.Valid() != nil {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey is invalid. please login again")
	}
	if !verifyClaims(userClaims, in.GetUserId(), in.GetPartyId()) {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authkey does not match supplied data")
	}
	res, err := s.DB.Exec(placePosterQuery, in.GetPartyId(), in.GetUserId(), in.GetLocation().Lng, in.GetLocation().Lat)
	if err != nil {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to insert poster to database: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return &pb.PlacementResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to get posterId from query: %v", err)
	}
	return &pb.PlacementResponse{Code: pb.ResponseCode_OK, PosterId: int32(id)}, nil
}

// RemovePoster will attempt to remove a poster from the database at a specific location.
func (s *server) RemovePoster(ctx context.Context, in *pb.RemovePosterRequest) (*pb.RemovePosterResponse, error) {
	if in.GetUserId() == 0 {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("userId not set")
	}
	if in.GetLocation() == nil {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("poster location not set")
	}
	if in.GetPartyId() == 0 {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("partyId not set")
	}
	if in.GetAuthKey() == "" {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey not set")
	}
	// verify authkey
	userClaims := tokenService.ParseAccessToken(in.GetAuthKey())
	if userClaims == nil || userClaims.Valid() != nil {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey is invalid. please login again")
	}
	if !verifyClaims(userClaims, in.GetUserId(), in.GetPartyId()) {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authkey does not match supplied data")
	}

	// find poster belonging to party that is closest to location
	location := in.GetLocation()

	res, err := s.DB.Query(posterDistanceQuery, location.GetLng(), location.GetLat(), in.GetPartyId(), removePosterMaxDistance)
	if err != nil {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query posters %v", err)
	}

	// check that there was a row returned

	if !res.Next() {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("no posters found within %d meters", removePosterMaxDistance)
	}
	defer res.Close()
	var poster Poster
	err = res.Scan(&poster.posterId, &poster.distance)
	if err != nil {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to scan sql result: %v", err)
	}

	_, err = s.DB.Exec(removePosterQuery, in.GetUserId(), poster.posterId, in.GetPartyId())
	if err != nil {
		return &pb.RemovePosterResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to remove poster: %v", err)
	}
	return &pb.RemovePosterResponse{Code: pb.ResponseCode_OK, Posterid: poster.posterId}, nil
}

// RegisterAccount will create a new user account.
func (s *server) RegisterAccount(ctx context.Context, in *pb.RegisterAccountRequest) (*pb.RegisterAccountResponse, error) {
	if in.GetUsername() == "" {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("username can't be empty")
	}
	if in.GetFirstName() == "" {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("first name can't be empty")
	}
	if in.GetLastName() == "" {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("last name can't be empty")
	}
	if in.GetPassword() == "" {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("password can't be empty")
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to start transaction %v", err)
	}
	// check does the username already exist
	res, err := s.DB.Query(accountExistsQuery, in.GetUsername())
	defer res.Close()
	if err != nil {
		_ = tx.Rollback()
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query username from database: %v", err)
	}
	if res.Next() {
		_ = tx.Rollback()
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("username already exists")
	}
	pwhash, err := hash(in.GetPassword())
	if err != nil {
		_ = tx.Rollback()
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to create password hash: %v", err)
	}
	// add acount to users table
	addAccountRes, err := tx.Exec(registerAccountQuery, in.GetUsername(), pwhash)
	if err != nil {
		_ = tx.Rollback()
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to add account to database: %v", err)
	}
	userId, err := addAccountRes.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to read userid from database: %v", err)
	}
	// add account to userinfo table
	_, err = tx.Exec(addUserinfoQuery, userId, in.GetFirstName(), in.GetLastName())
	if err != nil {
		_ = tx.Rollback()
		return &pb.RegisterAccountResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to add account to userinfo database: %v", err)
	}
	_ = tx.Commit()
	return &pb.RegisterAccountResponse{Code: pb.ResponseCode_OK}, nil
}

// LoginAccount will allow a user to login using their username and password.
func (s *server) LoginAccount(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	if in.GetUsername() == "" {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("username not supplied")
	}
	if in.GetPassword() == "" {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("password not supplied")
	}

	res, err := s.DB.Query(userInfoQuery, in.GetUsername())
	if err != nil {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query database for username %v. err: %v", in.GetUsername(), err)
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
	defer res.Close()
	if err := bcrypt.CompareHashAndPassword([]byte(result.Pwhash), []byte(in.GetPassword())); err != nil {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to login")
	}

	// generate JWT here and send back in authkey field
	claims := tokenService.UserClaims{
		UserID:   int32(result.UserId),
		Username: result.Username,
		PartyId:  int32(result.PartyId),
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
		},
	}
	accessToken, err := tokenService.NewAccessToken(claims)
	if err != nil {
		return &pb.LoginResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to create access token %v", err)
	}

	return &pb.LoginResponse{AuthKey: accessToken, Code: pb.ResponseCode_OK, Party: result.PartyName, UserId: int32(result.UserId), PartyId: int32(result.PartyId)}, nil
}

// RetrieveUpdates will request all changes to the poster database that have been made since
// the time specified in the lastupdated field in the request
func (s *server) RetrieveUpdates(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	if in.GetAuthKey() == "" {
		return &pb.UpdateResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("no authkey provided")
	}
	if in.GetUserId() == 0 {
		return &pb.UpdateResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("no userId provided")
	}
	if in.GetPartyid() == 0 {
		return &pb.UpdateResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("no partyId provided")
	}
	if in.GetLastUpdated() == nil {
		return &pb.UpdateResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("no lastUpdated provided")
	}

	// verify authkey
	userClaims := tokenService.ParseAccessToken(in.GetAuthKey())
	if userClaims == nil || userClaims.Valid() != nil {
		return &pb.UpdateResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey is invalid. please login again")
	}
	if !verifyClaims(userClaims, in.GetUserId(), in.GetPartyid()) {
		return &pb.UpdateResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authkey does not match supplied data")
	}

	rows, err := s.DB.Query("select posterID, partyId, userID, removed, st_y(location) as latitude, st_x(location) as longitude from fyp_schema.posters where partyId = ? and updated > from_unixtime(?)", in.Partyid, in.LastUpdated.AsTime().Unix())
	if err != nil {
		return &pb.UpdateResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to query database for posters %v", err)
	}
	defer rows.Close()
	var posters []*pb.Poster
	for rows.Next() {
		var poster PosterUpdate
		removed := []uint8{}
		t := false
		err = rows.Scan(&poster.PosterId, &poster.PartyId, &poster.UserID, &removed, &poster.location.Lat, &poster.location.Lng)
		if err != nil {
			return &pb.UpdateResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to get updates %v", err)
		}
		if removed != nil {
			t = true
		}
		posters = append(posters, &pb.Poster{PlacedBy: poster.UserID, Party: poster.PartyId, Posterid: poster.PosterId, Location: &poster.location, Removed: t})
	}
	return &pb.UpdateResponse{Posters: posters, Code: pb.ResponseCode_OK}, nil
}

// RetrieveParties will return a list of all parties available for a user to join
func (s *server) RetrieveParties(ctx context.Context, in *pb.RetrievePartiesRequest) (*pb.RetrievePartiesResponse, error) {
	if in.GetAuthKey() == "" {
		return &pb.RetrievePartiesResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("auth key is empty")
	}
	// verify authkey
	userClaims := tokenService.ParseAccessToken(in.GetAuthKey())
	if userClaims == nil || userClaims.Valid() != nil {
		return &pb.RetrievePartiesResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("authKey is invalid. please login again")
	}

	rows, err := s.DB.Query("select partyID, partyName from fyp_schema.parties where partyID > 1")
	if err != nil {
		return &pb.RetrievePartiesResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to retrieve party list from database %v", err)
	}
	var parties []*pb.Party
	for rows.Next() {
		party := pb.Party{}
		err = rows.Scan(&party.PartyID, &party.Name)
		if err != nil {
			return &pb.RetrievePartiesResponse{Code: pb.ResponseCode_FAILED}, fmt.Errorf("failed to get party list: %v", err)
		}
		parties = append(parties, &party)
	}

	return &pb.RetrievePartiesResponse{Code: pb.ResponseCode_OK, Parties: parties}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("192.168.0.194:%d", *port))
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
