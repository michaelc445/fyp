package com.example.myapplication.data;

import com.example.myapplication.Data;
import com.example.myapplication.data.model.LoggedInUser;
import com.michaelc445.messages.LoginRequest;
import com.michaelc445.messages.LoginResponse;
import com.michaelc445.messages.PosterAppGrpc;
import com.michaelc445.messages.ResponseCode;

import java.io.IOException;
import java.util.concurrent.TimeUnit;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

/**
 * Class that handles authentication w/ login credentials and retrieves user information.
 */
public class LoginDataSource {

    public Result<LoggedInUser> login(String username, String password) {

        try {

            ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();

            PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);

            LoginRequest req = LoginRequest.newBuilder()
                    .setUsername(username)
                    .setPassword(password).build();

            LoginResponse res = bStub.loginAccount(req);
            if (res.getCode() == ResponseCode.FAILED){
                return new Result.Error(new IOException("Error logging in"));
            }
            mChannel.shutdown();
            LoggedInUser newUser =
                    new LoggedInUser(
                            res.getUserId(),
                            username,
                            res.getAuthKey(),
                            res.getPartyId(),
                            res.getParty());
            return new Result.Success<>(newUser);
        } catch (Exception e) {
            return new Result.Error(new IOException("Error logging in", e));
        }
    }

    public void logout() {
        // TODO: revoke authentication
    }
}