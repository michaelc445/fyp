package com.example.myapplication;

import com.example.myapplication.data.model.LoggedInUser;

public class UserData {
    private static UserData instance;
    private static LoggedInUser user;

    UserData() {}

    public static synchronized UserData getInstance() {
        if (instance == null) {
            instance = new UserData();
        }
        return instance;
    }

    public static LoggedInUser getUser() {
        return user;
    }

    public static void setUser(LoggedInUser newUser) {
        user = newUser;
    }
}
