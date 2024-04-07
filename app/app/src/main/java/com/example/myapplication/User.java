package com.example.myapplication;

public class User {

    private final String firstName;
    private final String lastName;

    private final int userID;

    public User(int userID, String firstName, String lastName){
        this.firstName = firstName;
        this.lastName = lastName;
        this.userID = userID;
    }

    public String getFirstName(){
        return this.firstName;
    }
    public String getLastName(){
        return this.lastName;
    }

    public int getUserID(){
        return this.userID;
    }

}
