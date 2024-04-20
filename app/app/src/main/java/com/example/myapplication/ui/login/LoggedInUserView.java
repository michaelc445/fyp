package com.example.myapplication.ui.login;

import com.example.myapplication.data.model.LoggedInUser;

/**
 * Class exposing authenticated user details to the UI.
 */
class LoggedInUserView {
    private final LoggedInUser user;
    //... other data fields that may be accessible to the UI

    LoggedInUserView(LoggedInUser user) {
        this.user = user;
    }

    String getDisplayName() {
        return user.getUserName();
    }
    LoggedInUser getUser(){
        return this.user;
    }
}