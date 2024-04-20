package com.example.myapplication;

public class Party {

    private final String partyName;
    private final int partyID;

    public Party(String partyName, int partyID){
        this.partyID = partyID;
        this.partyName = partyName;
    }
    public String getPartyName(){
        return this.partyName;
    }
    public int getPartyID(){
        return this.partyID;
    }
}
