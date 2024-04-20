package com.example.myapplication.data.model;

import java.io.Serializable;

/**
 * Data class that captures user information for logged in users retrieved from LoginRepository
 */
public class LoggedInUser implements Serializable {

    private final int userId;
    private final String userName;

    private final String authKey;
    private final int partyId;
    private final String partyName;

    public LoggedInUser(int userId, String userName, String authKey,int partyId,String partyName) {
        this.userId = userId;
        this.userName = userName;
        this.authKey = authKey;
        this.partyId = partyId;
        this.partyName = partyName;
    }

    public int getUserId() {
        return this.userId;
    }

    public String getUserName() {
        return userName;
    }

    public String getAuthKey() {
        return authKey;
    }

    public int getPartyId() {
        return partyId;
    }



    public String getPartyName() {
        return partyName;
    }

    public String toString(){
        return "id: "+this.userId+ "username: "+this.userName;
    }
}