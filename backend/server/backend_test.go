package main

import (
	"context"
	"database/sql/driver"
	"log"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	pb "github.com/michaelc445/proto"
)

func TestRemovePoster(t *testing.T) {
	tests := []struct {
		name         string
		userId       int32
		partyId      int32
		location     *pb.Location
		posterId     int32
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
			mock.ExpectExec("update").WithArgs(tc.posterId, tc.partyId).WillReturnResult(sqlmock.NewResult(0, 0))

			res, err := server.RemovePoster(ctx, &pb.RemovePosterRequest{UserId: tc.userId, PartyId: tc.partyId, Location: tc.location})

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
			returnResult: sqlmock.NewResult(1, 2),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
		},
		{
			name:         "partyId not set",
			userId:       1,
			location:     &pb.Location{Lat: 1, Lng: 2},
			returnResult: sqlmock.NewResult(1, 2),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
		},
		{
			name:         "Location not set",
			userId:       1,
			partyId:      1,
			returnResult: sqlmock.NewResult(1, 2),
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
			mock.ExpectExec("insert").WithArgs(tc.partyId, tc.userId, tc.location.GetLat(), tc.location.GetLng()).WillReturnResult(tc.returnResult)

			res, err := server.PlacePoster(ctx, &pb.PlacementRequest{UserId: tc.userId, PartyId: tc.partyId, Location: tc.location})

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
			mock.ExpectQuery("select").WithArgs(tc.username).WillReturnRows(tc.accountExistsResult)
			mock.ExpectExec("insert").WithArgs(tc.username, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 2))
			mock.ExpectExec("insert").WithArgs(1, tc.firstName, tc.lastName).WillReturnResult(sqlmock.NewResult(1, 2))
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
