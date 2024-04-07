package main

import (
	"context"
	"database/sql/driver"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt"
	"github.com/michaelc445/fyp/tokenService"

	pb "github.com/michaelc445/proto"
)

func TestOutstandingPosters(t *testing.T) {

	tests := []struct {
		name         string
		userId       int32
		partyId      int32
		posterRows   *sqlmock.Rows
		electionDate time.Time
		wantErr      bool
		wantRes      *pb.PosterTimeResponse
	}{
		{
			name:    "userId not set",
			partyId: 1,
			posterRows: sqlmock.NewRows([]string{"created", "posterId", "userID", "username", "firstName", "lastName"}).
				AddRow(time.Now().Unix(), 1, 1, "michael1234", "Michael", "test1").
				AddRow(time.Now().Unix(), 2, 2, "michael1235", "Michael", "test2").
				AddRow(time.Now().Unix(), 3, 3, "michael1236", "Michael", "test3").
				AddRow(time.Now().Unix(), 4, 4, "michael1237", "Michael", "test4"),

			electionDate: time.Now(),
			wantErr:      true,
			wantRes:      &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:   "partyId not set",
			userId: 1,
			posterRows: sqlmock.NewRows([]string{"created", "posterId", "userID", "username", "firstName", "lastName"}).
				AddRow(time.Now().Unix(), 1, 1, "michael1234", "Michael", "test1").
				AddRow(time.Now().Unix(), 2, 2, "michael1235", "Michael", "test2").
				AddRow(time.Now().Unix(), 3, 3, "michael1236", "Michael", "test3").
				AddRow(time.Now().Unix(), 4, 4, "michael1237", "Michael", "test4"),

			electionDate: time.Now(),
			wantErr:      true,
			wantRes:      &pb.PosterTimeResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:    "success",
			partyId: 1,
			userId:  1,
			posterRows: sqlmock.NewRows([]string{"created", "posterId", "userID", "username", "firstName", "lastName"}).
				AddRow(time.Now().Unix(), 1, 1, "michael1234", "Michael", "test1").
				AddRow(time.Now().Unix(), 2, 2, "michael1235", "Michael", "test2").
				AddRow(time.Now().Unix(), 3, 3, "michael1236", "Michael", "test3").
				AddRow(time.Now().Unix(), 4, 4, "michael1237", "Michael", "test4"),

			electionDate: time.Now(),
			wantErr:      false,
			wantRes: &pb.PosterTimeResponse{
				Code: pb.ResponseCode_OK,
				Posters: []*pb.PosterUser{
					{Poster: &pb.Poster{PlacedBy: 1, Posterid: 1}, Username: "michael1234", FirstName: "Michael", LastName: "test1", Created: timestamppb.Now()},
					{Poster: &pb.Poster{PlacedBy: 2, Posterid: 2}, Username: "michael1235", FirstName: "Michael", LastName: "test2", Created: timestamppb.Now()},
					{Poster: &pb.Poster{PlacedBy: 3, Posterid: 3}, Username: "michael1236", FirstName: "Michael", LastName: "test3", Created: timestamppb.Now()},
					{Poster: &pb.Poster{PlacedBy: 4, Posterid: 4}, Username: "michael1237", FirstName: "Michael", LastName: "test4", Created: timestamppb.Now()},
				},
				RemovalDate: timestamppb.Now(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}

			mock.ExpectQuery("select").WithArgs(tc.partyId).WillReturnRows(tc.posterRows)
			electionRows := sqlmock.NewRows([]string{"electionDate"}).AddRow(tc.electionDate.Unix())
			mock.ExpectQuery("select").WithArgs(tc.partyId).WillReturnRows(electionRows)
			userClaims := tokenService.UserClaims{
				UserID:   tc.userId,
				Username: "test",
				PartyId:  tc.partyId,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
				},
			}

			authKey, err := tokenService.NewAccessToken(userClaims)
			if err != nil {
				t.Fatalf("failed to create jwt: %v", err)
			}
			res, err := server.OutstandingPosters(ctx, &pb.PosterTimeRequest{
				UserId:  tc.userId,
				AuthKey: authKey,
				PartyId: tc.partyId,
			})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}
			if len(res.GetPosters()) != len(tc.wantRes.GetPosters()) {
				t.Fatalf("expected posters of length %v got posters of length %v", len(tc.wantRes.GetPosters()), len(res.GetPosters()))
			}

			for i, val := range res.GetPosters() {
				if tc.wantRes.Posters[i].FirstName != val.FirstName {
					t.Fatalf("expected poster %v got poster %v", tc.wantRes.Posters[i], res.Posters[i])
				}
				if tc.wantRes.Posters[i].Username != val.Username {
					t.Fatalf("expected poster %v got poster %v", tc.wantRes.Posters[i], res.Posters[i])
				}
				if tc.wantRes.Posters[i].LastName != val.LastName {
					t.Fatalf("expected poster %v got poster %v", tc.wantRes.Posters[i], res.Posters[i])
				}
			}

		})
	}
}

func TestNewElection(t *testing.T) {
	tests := []struct {
		name         string
		userId       int32
		partyId      int32
		queryRows    *sqlmock.Rows
		execRes      driver.Result
		startDate    *timestamppb.Timestamp
		electionDate *timestamppb.Timestamp
		wantErr      bool
		wantRes      *pb.CreateElectionResponse
	}{
		{
			name:         "userId not set",
			partyId:      1,
			queryRows:    sqlmock.NewRows([]string{"partyId", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			execRes:      sqlmock.NewResult(1, 1),
			wantErr:      true,
			startDate:    timestamppb.New(time.Now().Add(time.Hour)),
			electionDate: timestamppb.New(time.Now().Add(time.Hour + time.Hour*24*7)),
			wantRes:      &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:         "partyId not set",
			userId:       1,
			queryRows:    sqlmock.NewRows([]string{"partyId", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			execRes:      sqlmock.NewResult(1, 1),
			wantErr:      true,
			startDate:    timestamppb.New(time.Now().Add(time.Hour)),
			electionDate: timestamppb.New(time.Now().Add(time.Hour + time.Hour*24*7)),
			wantRes:      &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:         "election in past",
			partyId:      1,
			userId:       1,
			queryRows:    sqlmock.NewRows([]string{"partyId", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			execRes:      sqlmock.NewResult(1, 1),
			wantErr:      true,
			startDate:    timestamppb.New(time.Now().Add(-time.Hour * 24 * 8)),
			electionDate: timestamppb.New(time.Now().Add(-time.Hour * 24 * 7)),
			wantRes:      &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:         "start date after election date",
			partyId:      1,
			userId:       1,
			queryRows:    sqlmock.NewRows([]string{"partyId", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			execRes:      sqlmock.NewResult(1, 1),
			wantErr:      true,
			startDate:    timestamppb.New(time.Now().Add(time.Hour * 24 * 8)),
			electionDate: timestamppb.New(time.Now().Add(time.Hour * 24 * 7)),
			wantRes:      &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:         "user is not admin of party",
			partyId:      1,
			userId:       1,
			queryRows:    sqlmock.NewRows([]string{"partyId", "partyName", "admin"}),
			execRes:      sqlmock.NewResult(1, 1),
			wantErr:      true,
			startDate:    timestamppb.New(time.Now().Add(time.Hour)),
			electionDate: timestamppb.New(time.Now().Add(time.Hour + time.Hour*24*7)),
			wantRes:      &pb.CreateElectionResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:         "success",
			partyId:      1,
			userId:       1,
			queryRows:    sqlmock.NewRows([]string{"partyId", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			execRes:      sqlmock.NewResult(1, 1),
			wantErr:      false,
			startDate:    timestamppb.New(time.Now().Add(time.Hour)),
			electionDate: timestamppb.New(time.Now().Add(time.Hour + time.Hour*24*7)),
			wantRes:      &pb.CreateElectionResponse{Code: pb.ResponseCode_OK},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}

			mock.ExpectQuery("select").WithArgs(tc.partyId, tc.userId).WillReturnRows(tc.queryRows)
			mock.ExpectExec("replace").WithArgs(tc.partyId, tc.startDate.AsTime().Unix(), tc.electionDate.AsTime().Unix()).WillReturnResult(tc.execRes)
			userClaims := tokenService.UserClaims{
				UserID:   tc.userId,
				Username: "test",
				PartyId:  tc.partyId,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
				},
			}
			authKey, err := tokenService.NewAccessToken(userClaims)
			if err != nil {
				t.Fatalf("failed to create jwt: %v", err)
			}
			res, err := server.NewElection(ctx, &pb.CreateElectionRequest{
				UserId:       tc.userId,
				AuthKey:      authKey,
				PartyId:      tc.partyId,
				StartDate:    tc.startDate,
				ElectionDate: tc.electionDate,
			})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if !proto.Equal(res, tc.wantRes) {
				t.Fatalf("got response %v want response %v", res, tc.wantRes)
			}

		})
	}
}

func TestRetrieveProfileStats(t *testing.T) {
	tests := []struct {
		name      string
		userId    int32
		partyId   int32
		queryRows *sqlmock.Rows
		wantErr   bool
		wantRes   *pb.ProfileResponse
	}{
		{
			name:      "userId not set",
			partyId:   1,
			queryRows: sqlmock.NewRows([]string{"placed", "removed", "partyName"}).AddRow(1, 1, "fake_party"),
			wantErr:   true,
			wantRes:   &pb.ProfileResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:      "partyId not set",
			userId:    1,
			queryRows: sqlmock.NewRows([]string{"placed", "removed", "partyName"}).AddRow(1, 1, "fake_party"),
			wantErr:   true,
			wantRes:   &pb.ProfileResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:      "success",
			partyId:   1,
			userId:    1,
			queryRows: sqlmock.NewRows([]string{"placed", "removed", "partyName"}).AddRow(3, 4, "fake_party"),
			wantErr:   false,
			wantRes:   &pb.ProfileResponse{Code: pb.ResponseCode_OK, PlacedPosters: 3, RemovedPosters: 4, PartyName: "fake_party"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}

			mock.ExpectQuery("select").WithArgs(tc.userId, tc.userId, tc.partyId).WillReturnRows(tc.queryRows)

			userClaims := tokenService.UserClaims{
				UserID:   tc.userId,
				Username: "test",
				PartyId:  tc.partyId,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
				},
			}
			authKey, err := tokenService.NewAccessToken(userClaims)
			if err != nil {
				t.Fatalf("failed to create jwt: %v", err)
			}
			res, err := server.RetrieveProfileStats(ctx, &pb.ProfileRequest{UserId: tc.userId, AuthKey: authKey, PartyId: tc.partyId})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if !proto.Equal(res, tc.wantRes) {
				t.Fatalf("got response %v want response %v", res, tc.wantRes)
			}

		})
	}
}

func TestApproveMembers(t *testing.T) {
	tests := []struct {
		name            string
		userId          int32
		partyId         int32
		approvedMembers []*pb.Member
		deniedMembers   []*pb.Member
		adminRows       *sqlmock.Rows
		wantErr         bool
		wantCode        pb.ResponseCode
	}{
		{
			name:      "userId not set",
			userId:    0,
			partyId:   1,
			wantCode:  pb.ResponseCode_FAILED,
			adminRows: sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 0),
			wantErr:   true,
		},
		{
			name:      "partyId not set",
			userId:    1,
			partyId:   0,
			wantCode:  pb.ResponseCode_FAILED,
			adminRows: sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			wantErr:   true,
		},
		{
			name:      "user is not admin of party",
			userId:    1,
			partyId:   1,
			wantCode:  pb.ResponseCode_FAILED,
			adminRows: sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			wantErr:   true,
		},
		{
			name:      "approve members",
			userId:    1,
			partyId:   3,
			wantCode:  pb.ResponseCode_OK,
			adminRows: sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			approvedMembers: []*pb.Member{
				{UserId: 2, FirstName: "john", LastName: "murphy"},
				{UserId: 3, FirstName: "jack", LastName: "sparrow"},
				{UserId: 4, FirstName: "forrest", LastName: "gump"},
			},
			wantErr: false,
		},
		{
			name:      "deny members",
			userId:    1,
			partyId:   1,
			wantCode:  pb.ResponseCode_OK,
			adminRows: sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			deniedMembers: []*pb.Member{
				{UserId: 2, FirstName: "john", LastName: "murphy"},
				{UserId: 3, FirstName: "jack", LastName: "sparrow"},
				{UserId: 4, FirstName: "forrest", LastName: "gump"},
			},
			wantErr: false,
		},
		{
			name:      "approve and deny members",
			userId:    3,
			partyId:   1,
			wantCode:  pb.ResponseCode_OK,
			adminRows: sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			approvedMembers: []*pb.Member{
				{UserId: 2, FirstName: "john", LastName: "murphy"},
				{UserId: 3, FirstName: "jack", LastName: "sparrow"},
				{UserId: 4, FirstName: "forrest", LastName: "gump"},
			},
			deniedMembers: []*pb.Member{
				{UserId: 5, FirstName: "murphy", LastName: "john"},
				{UserId: 6, FirstName: "sparrow", LastName: "jack"},
				{UserId: 7, FirstName: "gump", LastName: "forrest"},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}
			mock.ExpectBegin()

			mock.ExpectQuery("select").WithArgs(tc.partyId, tc.userId).WillReturnRows(tc.adminRows)

			for _, member := range tc.approvedMembers {
				mock.ExpectQuery("select").
					WithArgs(member.GetUserId(), tc.partyId).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectExec("update fyp_schema.users").WithArgs(tc.partyId, member.GetUserId()).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("update fyp_schema.posters").WithArgs(tc.partyId, member.GetUserId()).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("update fyp_schema.joinRequests").WithArgs(member.GetUserId(), tc.partyId).WillReturnResult(sqlmock.NewResult(1, 1))
			}

			for _, member := range tc.deniedMembers {
				mock.ExpectExec("update").WithArgs(member.GetUserId(), tc.partyId).WillReturnResult(sqlmock.NewResult(1, 1))
			}
			mock.ExpectCommit()
			userClaims := tokenService.UserClaims{
				UserID:   tc.userId,
				Username: "test",
				PartyId:  tc.partyId,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
				},
			}
			authKey, err := tokenService.NewAccessToken(userClaims)
			if err != nil {
				t.Fatalf("failed to create jwt: %v", err)
			}
			res, err := server.ApproveMembers(ctx, &pb.ApproveMemberRequest{
				UserId:          tc.userId,
				AuthKey:         authKey,
				PartyId:         tc.partyId,
				ApprovedMembers: tc.approvedMembers,
				DeniedMembers:   tc.deniedMembers,
			})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if tc.wantCode != res.Code {
				t.Fatalf("got response code: %v want response code: %v", res.GetCode(), tc.wantCode)
			}

		})
	}
}

func TestRetrieveJoinRequests(t *testing.T) {
	tests := []struct {
		name            string
		userId          int32
		partyId         int32
		userAdminRows   *sqlmock.Rows
		joinRequestRows *sqlmock.Rows
		wantErr         bool
		wantRes         *pb.RetrieveJoinResponse
	}{
		{
			name:            "userId not set",
			userId:          0,
			partyId:         1,
			userAdminRows:   sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			joinRequestRows: sqlmock.NewRows([]string{"userID", "firstName", "lastName"}),
			wantRes:         &pb.RetrieveJoinResponse{Code: pb.ResponseCode_FAILED},
			wantErr:         true,
		},
		{
			name:            "partyId not set",
			userId:          1,
			partyId:         0,
			userAdminRows:   sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			joinRequestRows: sqlmock.NewRows([]string{"userID", "firstName", "lastName"}),
			wantRes:         &pb.RetrieveJoinResponse{Code: pb.ResponseCode_FAILED},
			wantErr:         true,
		},
		{
			name:            "user is not admin of party",
			userId:          1,
			partyId:         1,
			userAdminRows:   sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			joinRequestRows: sqlmock.NewRows([]string{"userID", "firstName", "lastName"}),
			wantRes:         &pb.RetrieveJoinResponse{Code: pb.ResponseCode_FAILED},
			wantErr:         true,
		},
		{
			name:            "no join requests",
			userId:          1,
			partyId:         1,
			userAdminRows:   sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			joinRequestRows: sqlmock.NewRows([]string{"userID", "firstName", "lastName"}),
			wantRes:         &pb.RetrieveJoinResponse{Code: pb.ResponseCode_OK, Members: []*pb.Member{}},
			wantErr:         false,
		},
		{
			name:          "success",
			userId:        2,
			partyId:       1,
			userAdminRows: sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			joinRequestRows: sqlmock.NewRows([]string{"userID", "firstName", "lastName"}).
				AddRow(2, "john", "murphy").
				AddRow(3, "jack", "sparrow").
				AddRow(4, "forrest", "gump"),

			wantRes: &pb.RetrieveJoinResponse{
				Code: pb.ResponseCode_OK,
				Members: []*pb.Member{
					{UserId: 2, FirstName: "john", LastName: "murphy"},
					{UserId: 3, FirstName: "jack", LastName: "sparrow"},
					{UserId: 4, FirstName: "forrest", LastName: "gump"},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}

			mock.ExpectQuery("select").WithArgs(tc.partyId, tc.userId).WillReturnRows(tc.userAdminRows)
			mock.ExpectQuery("select").WithArgs(tc.partyId).WillReturnRows(tc.joinRequestRows)

			userClaims := tokenService.UserClaims{
				UserID:   tc.userId,
				Username: "test",
				PartyId:  tc.partyId,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
				},
			}
			authKey, err := tokenService.NewAccessToken(userClaims)
			if err != nil {
				t.Fatalf("failed to create jwt: %v", err)
			}
			res, err := server.RetrieveJoinRequests(ctx, &pb.RetrieveJoinRequest{UserId: tc.userId, AuthKey: authKey, PartyId: tc.partyId})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if !proto.Equal(res, tc.wantRes) {
				t.Fatalf("got response %v want response %v", res, tc.wantRes)
			}

		})
	}
}

func TestJoinParty(t *testing.T) {
	tests := []struct {
		name                  string
		userId                int32
		partyId               int32
		userAdminRows         *sqlmock.Rows
		joinRequestExistsRows *sqlmock.Rows
		partyExistsRows       *sqlmock.Rows
		joinRequestResult     driver.Result
		wantErr               bool

		wantCode pb.ResponseCode
	}{
		{
			name:                  "userId not set",
			userId:                0,
			partyId:               1,
			userAdminRows:         sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			partyExistsRows:       sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 23),
			joinRequestExistsRows: sqlmock.NewRows([]string{"userID", "partyID", "reviewed"}),
			joinRequestResult:     sqlmock.NewResult(2, 1),
			wantCode:              pb.ResponseCode_FAILED,
			wantErr:               true,
		},
		{
			name:                  "partyId not set",
			userId:                1,
			partyId:               0,
			userAdminRows:         sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			partyExistsRows:       sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 23),
			joinRequestExistsRows: sqlmock.NewRows([]string{"userID", "partyID", "reviewed"}),
			joinRequestResult:     sqlmock.NewResult(2, 1),
			wantCode:              pb.ResponseCode_FAILED,
			wantErr:               true,
		},
		{
			name:                  "user is admin of party",
			userId:                1,
			partyId:               1,
			userAdminRows:         sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			partyExistsRows:       sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			joinRequestExistsRows: sqlmock.NewRows([]string{"userID", "partyID", "reviewed"}),
			joinRequestResult:     sqlmock.NewResult(2, 1),
			wantCode:              pb.ResponseCode_FAILED,
			wantErr:               true,
		},
		{
			name:                  "party does not exist",
			userId:                1,
			partyId:               1,
			userAdminRows:         sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			partyExistsRows:       sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			joinRequestExistsRows: sqlmock.NewRows([]string{"userID", "partyID", "reviewed"}),
			joinRequestResult:     sqlmock.NewResult(2, 1),
			wantCode:              pb.ResponseCode_FAILED,
			wantErr:               true,
		},
		{
			name:                  "join request already exists",
			userId:                1,
			partyId:               1,
			userAdminRows:         sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			partyExistsRows:       sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 1),
			joinRequestExistsRows: sqlmock.NewRows([]string{"userID", "partyID", "reviewed"}).AddRow(1, 1, 0),
			joinRequestResult:     sqlmock.NewResult(2, 1),
			wantCode:              pb.ResponseCode_FAILED,
			wantErr:               true,
		},
		{
			name:                  "success",
			userId:                2,
			partyId:               1,
			userAdminRows:         sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			partyExistsRows:       sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(1, "fake_party", 23),
			joinRequestExistsRows: sqlmock.NewRows([]string{"userID", "partyID", "reviewed"}),
			joinRequestResult:     sqlmock.NewResult(2, 1),
			wantErr:               false,
			wantCode:              pb.ResponseCode_OK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}

			mock.ExpectQuery("select").WithArgs(tc.partyId).WillReturnRows(tc.partyExistsRows)
			mock.ExpectQuery("select").WithArgs(tc.userId).WillReturnRows(tc.userAdminRows)
			mock.ExpectQuery("select").WithArgs(tc.userId, tc.partyId).WillReturnRows(tc.joinRequestExistsRows)

			mock.ExpectBegin()
			mock.ExpectExec("update").WithArgs(tc.userId).WillReturnResult(driver.ResultNoRows)
			mock.ExpectExec("insert").WithArgs(tc.userId, tc.partyId).WillReturnResult(tc.joinRequestResult)
			mock.ExpectCommit()

			userClaims := tokenService.UserClaims{
				UserID:   tc.userId,
				Username: "test",
				PartyId:  tc.partyId,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
				},
			}
			authKey, err := tokenService.NewAccessToken(userClaims)
			if err != nil {
				t.Fatalf("failed to create jwt: %v", err)
			}
			res, err := server.JoinParty(ctx, &pb.JoinPartyRequest{UserId: tc.userId, AuthKey: authKey, PartyId: tc.partyId})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if res.Code != tc.wantCode {
				t.Fatalf("got code %v want code %v", res.Code, tc.wantCode)
			}

		})
	}
}

func TestRegisterParty(t *testing.T) {
	tests := []struct {
		name              string
		userId            int32
		partyId           int32
		newPartyId        int32
		partyName         string
		userAdminRows     *sqlmock.Rows
		partyExistsRows   *sqlmock.Rows
		updateUserResult  driver.Result
		createPartyResult driver.Result
		userClaims        *tokenService.UserClaims
		wantErr           bool

		wantCode pb.ResponseCode
	}{
		{
			name:              "partyName not set",
			userId:            1,
			partyId:           1,
			newPartyId:        2,
			partyName:         "",
			userAdminRows:     sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			partyExistsRows:   sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			updateUserResult:  sqlmock.NewResult(0, 1),
			createPartyResult: sqlmock.NewResult(2, 1),
			wantCode:          pb.ResponseCode_FAILED,
			wantErr:           true,
		},
		{
			name:              "userId not set",
			userId:            0,
			partyId:           1,
			newPartyId:        2,
			partyName:         "hello",
			userAdminRows:     sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			partyExistsRows:   sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			updateUserResult:  sqlmock.NewResult(0, 1),
			createPartyResult: sqlmock.NewResult(2, 1),
			wantCode:          pb.ResponseCode_FAILED,
			wantErr:           true,
		},
		{
			name:              "user is admin  of party",
			userId:            1,
			partyId:           1,
			newPartyId:        2,
			partyName:         "new_party",
			userAdminRows:     sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(2, "haha", 1),
			partyExistsRows:   sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			updateUserResult:  sqlmock.NewResult(0, 0),
			createPartyResult: sqlmock.NewResult(0, 0),
			wantCode:          pb.ResponseCode_FAILED,
			wantErr:           true,
		},
		{
			name:              "party already exists",
			userId:            1,
			partyId:           1,
			newPartyId:        2,
			partyName:         "new_party",
			userAdminRows:     sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			partyExistsRows:   sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}).AddRow(2, "haha", 1),
			updateUserResult:  sqlmock.NewResult(0, 0),
			createPartyResult: sqlmock.NewResult(0, 0),
			wantCode:          pb.ResponseCode_FAILED,
			wantErr:           true,
		},
		{
			name:              "success",
			userId:            2,
			partyId:           1,
			newPartyId:        2,
			partyName:         "new party",
			userAdminRows:     sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			partyExistsRows:   sqlmock.NewRows([]string{"partyID ", "partyName", "admin"}),
			updateUserResult:  sqlmock.NewResult(2, 1),
			createPartyResult: sqlmock.NewResult(2, 1),
			wantErr:           false,
			wantCode:          pb.ResponseCode_OK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}
			mock.ExpectBegin()

			mock.ExpectQuery("select").WithArgs(tc.partyId, tc.userId).WillReturnRows(tc.userAdminRows)
			mock.ExpectQuery("select").WithArgs(tc.partyName).WillReturnRows(tc.partyExistsRows)
			mock.ExpectExec("insert").WithArgs(tc.partyName, tc.userId).WillReturnResult(tc.createPartyResult)
			mock.ExpectExec("update").WithArgs(tc.newPartyId, tc.userId).WillReturnResult(tc.updateUserResult)

			mock.ExpectCommit()
			userClaims := tokenService.UserClaims{
				UserID:   tc.userId,
				Username: "test",
				PartyId:  tc.partyId,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
				},
			}
			authKey, err := tokenService.NewAccessToken(userClaims)
			if err != nil {
				t.Fatalf("failed to create jwt: %v", err)
			}
			res, err := server.RegisterParty(ctx, &pb.RegisterPartyRequest{UserId: tc.userId, AuthKey: authKey, PartyName: tc.partyName})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if res.Code != tc.wantCode {
				t.Fatalf("got code %v want code %v", res.Code, tc.wantCode)
			}
			authKey = res.GetAuthKey()
			if authKey != "" {
				newClaims := tokenService.ParseAccessToken(authKey)
				if newClaims.PartyId != tc.newPartyId {
					t.Fatalf("expected partyid: %v got partyID: %v", tc.newPartyId, newClaims.PartyId)
				}
			}

		})
	}
}

func TestRemovePoster(t *testing.T) {
	tests := []struct {
		name         string
		userId       int32
		partyId      int32
		location     *pb.Location
		posterId     int32
		userClaims   *tokenService.UserClaims
		wantPosterId int32
		wantErr      bool
		returnRows   *sqlmock.Rows
		wantCode     pb.ResponseCode
	}{
		{
			name:         "poster does not exist",
			userId:       1,
			partyId:      1,
			location:     &pb.Location{Lat: 1.0, Lng: 1.0},
			posterId:     1,
			wantPosterId: 0,
			returnRows:   sqlmock.NewRows([]string{"posterId", "distance"}),
			wantCode:     pb.ResponseCode_FAILED,
			wantErr:      true,
		},
		{
			name:         "poster does not belong to party",
			userId:       1,
			partyId:      1,
			location:     &pb.Location{Lat: 1, Lng: 1},
			posterId:     1,
			wantPosterId: 0,
			returnRows:   sqlmock.NewRows([]string{"posterId", "distance"}).AddRow(2, 1),
			wantCode:     pb.ResponseCode_FAILED,
			wantErr:      true,
		},
		{
			name:         "userId not set",
			partyId:      1,
			location:     &pb.Location{Lat: 1, Lng: 1},
			posterId:     1,
			wantPosterId: 0,
			returnRows:   sqlmock.NewRows([]string{"posterId", "distance"}).AddRow(1, 1),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
		},
		{
			name:         "partyId not set",
			userId:       1,
			location:     &pb.Location{Lat: 1, Lng: 1},
			posterId:     1,
			wantPosterId: 0,
			returnRows:   sqlmock.NewRows([]string{"posterId", "distance"}).AddRow(1, 1),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
		},
		{
			name:         "location not set",
			userId:       1,
			partyId:      1,
			posterId:     1,
			wantPosterId: 0,
			returnRows:   sqlmock.NewRows([]string{"posterId", "distance"}).AddRow(1, 1),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
		},
		{
			name:         "success",
			userId:       1,
			partyId:      1,
			location:     &pb.Location{Lat: 1, Lng: 1},
			posterId:     1,
			wantPosterId: 1,
			returnRows:   sqlmock.NewRows([]string{"posterId", "distance"}).AddRow(1, 1),
			wantErr:      false,
			wantCode:     pb.ResponseCode_OK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}
			mock.ExpectQuery("select").WithArgs(tc.location.GetLat(), tc.location.GetLng(), tc.partyId, removePosterMaxDistance).WillReturnRows(tc.returnRows)
			mock.ExpectExec("update").WithArgs(tc.userId, tc.posterId, tc.partyId).WillReturnResult(sqlmock.NewResult(0, 0))

			userClaims := tokenService.UserClaims{
				UserID:   tc.userId,
				Username: "test",
				PartyId:  tc.partyId,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
				},
			}
			authKey, err := tokenService.NewAccessToken(userClaims)
			if err != nil {
				t.Fatalf("failed to create jwt: %v", err)
			}
			res, err := server.RemovePoster(ctx, &pb.RemovePosterRequest{UserId: tc.userId, PartyId: tc.partyId, Location: tc.location, AuthKey: authKey})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if res.Code != tc.wantCode {
				t.Fatalf("got code %v want code %v", res.Code, tc.wantCode)
			}
			if res.Posterid != tc.wantPosterId {
				t.Fatalf("got posterid %v want posterid %v", res.Posterid, tc.wantPosterId)
			}
		})
	}
}

func TestPlacePoster(t *testing.T) {
	tests := []struct {
		name         string
		userId       int32
		partyId      int32
		location     *pb.Location
		wantErr      bool
		returnResult driver.Result
		wantCode     pb.ResponseCode
	}{
		{
			name:         "userId not set",
			partyId:      1,
			location:     &pb.Location{Lat: 1, Lng: 2},
			returnResult: sqlmock.NewResult(0, 0),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
		},
		{
			name:         "partyId not set",
			userId:       1,
			location:     &pb.Location{Lat: 1, Lng: 2},
			returnResult: sqlmock.NewResult(0, 0),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
		},
		{
			name:         "Location not set",
			userId:       1,
			partyId:      1,
			returnResult: sqlmock.NewResult(0, 0),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
		},
		{
			name:         "success",
			userId:       1,
			partyId:      1,
			location:     &pb.Location{Lat: 1, Lng: 2},
			returnResult: sqlmock.NewResult(1, 2),
			wantErr:      false,
			wantCode:     pb.ResponseCode_OK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}
			mock.ExpectExec("insert").WithArgs(tc.partyId, tc.userId, tc.location.GetLng(), tc.location.GetLat()).WillReturnResult(tc.returnResult)

			userClaims := tokenService.UserClaims{
				UserID:   tc.userId,
				Username: "test",
				PartyId:  tc.partyId,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
				},
			}
			authKey, err := tokenService.NewAccessToken(userClaims)
			if err != nil {
				t.Fatalf("failed to create jwt: %v", err)
			}

			res, err := server.PlacePoster(ctx, &pb.PlacementRequest{UserId: tc.userId, PartyId: tc.partyId, Location: tc.location, AuthKey: authKey})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if res.Code != tc.wantCode {
				t.Fatalf("got code %v want code %v", res.Code, tc.wantCode)
			}
		})
	}
}

func TestRegisterAccount(t *testing.T) {
	tests := []struct {
		name                string
		username            string
		firstName           string
		lastName            string
		password            string
		wantErr             bool
		returnResult        driver.Result
		accountExistsResult *sqlmock.Rows
		wantCode            pb.ResponseCode
	}{
		{
			name:                "username not set",
			username:            "",
			firstName:           "Michael",
			lastName:            "lastName",
			password:            "fakePassword",
			returnResult:        sqlmock.NewResult(1, 2),
			accountExistsResult: sqlmock.NewRows([]string{"username", "userId"}),
			wantErr:             true,
			wantCode:            pb.ResponseCode_FAILED,
		},
		{
			name:                "first name not set",
			username:            "test_username",
			firstName:           "",
			lastName:            "lastName",
			password:            "fakePassword",
			returnResult:        sqlmock.NewResult(1, 2),
			accountExistsResult: sqlmock.NewRows([]string{"username", "userId"}),
			wantErr:             true,
			wantCode:            pb.ResponseCode_FAILED,
		},
		{
			name:                "last name not set",
			username:            "test_username",
			firstName:           "Michael",
			lastName:            "",
			password:            "fakePassword",
			returnResult:        sqlmock.NewResult(1, 2),
			accountExistsResult: sqlmock.NewRows([]string{"username", "userId"}),
			wantErr:             true,
			wantCode:            pb.ResponseCode_FAILED,
		},
		{
			name:                "password not set",
			username:            "test_username",
			firstName:           "Michael",
			lastName:            "test_lastname",
			password:            "",
			returnResult:        sqlmock.NewResult(1, 2),
			accountExistsResult: sqlmock.NewRows([]string{"username", "userId"}),
			wantErr:             true,
			wantCode:            pb.ResponseCode_FAILED,
		},
		{
			name:                "username already exists",
			username:            "test_username",
			firstName:           "Michael",
			lastName:            "test_lastname",
			password:            "test_password",
			returnResult:        sqlmock.NewResult(1, 2),
			accountExistsResult: sqlmock.NewRows([]string{"username", "userId"}).AddRow("test_username", 1),
			wantErr:             true,
			wantCode:            pb.ResponseCode_FAILED,
		},
		{
			name:                "success",
			username:            "test_username",
			firstName:           "Michael",
			lastName:            "test_last_name",
			password:            "fakePassword",
			returnResult:        sqlmock.NewResult(1, 2),
			accountExistsResult: sqlmock.NewRows([]string{"username", "userId"}),
			wantErr:             false,
			wantCode:            pb.ResponseCode_OK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}
			mock.ExpectBegin()

			mock.ExpectQuery("select").WithArgs(tc.username).WillReturnRows(tc.accountExistsResult)
			mock.ExpectExec("insert").WithArgs(tc.username, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 2))
			mock.ExpectExec("insert").WithArgs(1, tc.firstName, tc.lastName).WillReturnResult(sqlmock.NewResult(1, 2))

			mock.ExpectCommit()
			res, err := server.RegisterAccount(ctx,
				&pb.RegisterAccountRequest{
					Username:  tc.username,
					Password:  tc.password,
					FirstName: tc.firstName,
					LastName:  tc.lastName,
				})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if res.Code != tc.wantCode {
				t.Fatalf("got code %v want code %v", res.Code, tc.wantCode)
			}
		})
	}
}

func TestLoginAccount(t *testing.T) {
	pwhash, err := hash("fakePassword")
	if err != nil {
		log.Fatalf("failed to create password hash")
	}
	tests := []struct {
		name        string
		username    string
		password    string
		wantErr     bool
		loginResult *sqlmock.Rows
		wantCode    pb.ResponseCode
	}{
		{
			name:        "username not set",
			username:    "",
			password:    "fakePassword",
			loginResult: sqlmock.NewRows([]string{"userID", "partyID", "username", "pwhash", "partyName"}).AddRow(1, 1, "test", "fake", "party"),
			wantErr:     true,
			wantCode:    pb.ResponseCode_FAILED,
		},
		{
			name:        "password not set",
			username:    "test_username",
			password:    "",
			loginResult: sqlmock.NewRows([]string{"userID", "partyID", "username", "pwhash", "partyName"}).AddRow(1, 1, "test", "fake", "party"),
			wantErr:     true,
			wantCode:    pb.ResponseCode_FAILED,
		},
		{
			name:        "username doesn't exist",
			username:    "test_username",
			password:    "test_password",
			loginResult: sqlmock.NewRows([]string{"userID", "partyID", "username", "pwhash", "partyName"}),
			wantErr:     true,
			wantCode:    pb.ResponseCode_FAILED,
		},
		{
			name:        "success",
			username:    "test_username",
			password:    "fakePassword",
			loginResult: sqlmock.NewRows([]string{"userID", "partyID", "username", "pwhash", "partyName"}).AddRow(1, 1, "test_username", pwhash, "party"),
			wantErr:     false,
			wantCode:    pb.ResponseCode_OK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)
			}
			server := &server{DB: db}
			mock.ExpectQuery("select").WithArgs(tc.username).WillReturnRows(tc.loginResult)

			res, err := server.LoginAccount(ctx,
				&pb.LoginRequest{
					Username: tc.username,
					Password: tc.password,
				})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if res.Code != tc.wantCode {
				t.Fatalf("got code %v want code %v", res.Code, tc.wantCode)
			}
		})
	}
}

func Test_RetrieveUpdates(t *testing.T) {

	tests := []struct {
		name         string
		userId       int32
		partyId      int32
		lastUpdated  *timestamp.Timestamp
		wantErr      bool
		returnRows   *sqlmock.Rows
		wantCode     pb.ResponseCode
		wantResponse *pb.UpdateResponse
	}{
		{
			name:         "userId not set",
			partyId:      1,
			returnRows:   sqlmock.NewRows([]string{"posterID", "partyId", "userID", "removed", "latitude", "longitude"}),
			lastUpdated:  timestamppb.New(time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC)),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
			wantResponse: &pb.UpdateResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:         "partyId not set",
			userId:       1,
			returnRows:   sqlmock.NewRows([]string{"posterID", "partyId", "userID", "removed", "latitude", "longitude"}),
			lastUpdated:  timestamppb.New(time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC)),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
			wantResponse: &pb.UpdateResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:         "lastUpdated  not set",
			userId:       1,
			partyId:      1,
			returnRows:   sqlmock.NewRows([]string{"posterID", "partyId", "userID", "removed", "latitude", "longitude"}),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
			wantResponse: &pb.UpdateResponse{Code: pb.ResponseCode_FAILED},
		},
		{
			name:    "success",
			userId:  1,
			partyId: 1,
			returnRows: sqlmock.NewRows([]string{"posterID", "partyId", "userID", "removed", "latitude", "longitude"}).
				AddRow(1, 1, 1, nil, 1, 1).
				AddRow(2, 1, 1, nil, 2, 1),

			lastUpdated: timestamppb.New(time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC)),
			wantErr:     false,

			wantCode: pb.ResponseCode_OK,
			wantResponse: &pb.UpdateResponse{Code: pb.ResponseCode_OK, Posters: []*pb.Poster{
				{
					PlacedBy: 1,
					Party:    1,
					Posterid: 1,
					Location: &pb.Location{Lat: 1, Lng: 1},
				},
				{
					PlacedBy: 1,
					Party:    1,
					Posterid: 2,
					Location: &pb.Location{Lat: 2, Lng: 1},
				},
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}

			mock.ExpectQuery("select").WithArgs(tc.partyId, tc.lastUpdated.AsTime().Unix()).WillReturnRows(tc.returnRows)

			userClaims := tokenService.UserClaims{
				UserID:   tc.userId,
				Username: "test",
				PartyId:  tc.partyId,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
				},
			}
			authKey, err := tokenService.NewAccessToken(userClaims)
			if err != nil {
				t.Fatalf("failed to create jwt: %v", err)
			}

			res, err := server.RetrieveUpdates(ctx, &pb.UpdateRequest{Partyid: tc.partyId, UserId: tc.userId, AuthKey: authKey, LastUpdated: tc.lastUpdated})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}
			if !proto.Equal(res, tc.wantResponse) {
				t.Fatalf("got response %v want response %v", res, tc.wantResponse)
			}
		})
	}
}

func Test_RetrieveParties(t *testing.T) {
	tests := []struct {
		name        string
		userId      int32
		partyId     int32
		wantErr     bool
		invalidAuth bool
		returnRows  *sqlmock.Rows

		wantResponse *pb.RetrievePartiesResponse
	}{
		{
			name:         "invalid Authkey",
			userId:       1,
			partyId:      1,
			returnRows:   sqlmock.NewRows([]string{"partyID", "partyName"}),
			invalidAuth:  true,
			wantResponse: &pb.RetrievePartiesResponse{Code: pb.ResponseCode_FAILED},
			wantErr:      true,
		},
		{
			name:         "no parties",
			userId:       1,
			partyId:      1,
			returnRows:   sqlmock.NewRows([]string{"partyID", "partyName"}),
			wantResponse: &pb.RetrievePartiesResponse{Code: pb.ResponseCode_OK},
			invalidAuth:  false,
			wantErr:      false,
		},
		{
			name:        "1 party",
			partyId:     1,
			returnRows:  sqlmock.NewRows([]string{"partyID", "partyName"}).AddRow(1, "fake_party"),
			wantErr:     false,
			invalidAuth: false,
			wantResponse: &pb.RetrievePartiesResponse{Code: pb.ResponseCode_OK, Parties: []*pb.Party{
				{Name: "fake_party", PartyID: 1},
			}},
		},
		{
			name:   "multiple parties",
			userId: 1,
			returnRows: sqlmock.NewRows([]string{"partyID", "partyName"}).
				AddRow(1, "fake_party1").
				AddRow(2, "fake_party2").
				AddRow(3, "fake_party3"),
			wantErr: false,
			wantResponse: &pb.RetrievePartiesResponse{Code: pb.ResponseCode_OK, Parties: []*pb.Party{
				{Name: "fake_party1", PartyID: 1},
				{Name: "fake_party2", PartyID: 2},
				{Name: "fake_party3", PartyID: 3},
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}
			mock.ExpectQuery("select").WillReturnRows(tc.returnRows)

			userClaims := tokenService.UserClaims{
				UserID:   tc.userId,
				Username: "test",
				PartyId:  tc.partyId,
				StandardClaims: jwt.StandardClaims{
					IssuedAt:  time.Now().Unix(),
					ExpiresAt: time.Now().Add(time.Hour * 48).Unix(),
				},
			}
			authKey := ""
			if !tc.invalidAuth {
				authKey, err = tokenService.NewAccessToken(userClaims)
				if err != nil {
					t.Fatalf("failed to create jwt: %v", err)
				}
			}

			res, err := server.RetrieveParties(ctx, &pb.RetrievePartiesRequest{AuthKey: authKey})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if !proto.Equal(res, tc.wantResponse) {
				t.Fatalf("expected: %v\n got: %v", tc.wantResponse, res)
			}

		})
	}
}
