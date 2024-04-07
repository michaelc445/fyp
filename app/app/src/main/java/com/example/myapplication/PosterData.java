package com.example.myapplication;

import java.time.LocalDateTime;

public class PosterData {
    private final String username;
    private final String firstName;
    private final String lastName;
    private final LocalDateTime removalDate;
    private final int posterId;

    public PosterData(int posterId, String username, String firstName, String lastName, LocalDateTime removalDate) {
        this.posterId = posterId;
        this.username = username;
        this.firstName = firstName;
        this.lastName = lastName;
        this.removalDate = removalDate;
    }



    public int getPosterId() {
        return posterId;
    }

    public String getUsername() {
        return username;
    }

    public String getFirstName() {
        return firstName;
    }

    public String getLastName() {
        return lastName;
    }

    public LocalDateTime getRemovalDate() {
        return removalDate;
    }


}
