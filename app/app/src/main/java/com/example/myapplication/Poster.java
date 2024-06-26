package com.example.myapplication;

public class Poster {
    private final double latitude;
    private final double longitude;
    private final Integer id;
    private final boolean updated;
    private final boolean removed;

    public Poster(double lat, double lng, Integer id, boolean updated, boolean removed){
        this.longitude = lng;
        this.latitude = lat;
        this.id= id;
        this.removed = removed;
        this.updated = updated;
    }
    public double getLatitude(){
        return this.latitude;
    }
    public double getLongitude(){
        return this.longitude;
    }

    public int getId(){
        if (this.id != null){
            return this.id.intValue();
        }
        return 0;
    }
    public Integer getIdInteger(){
        return this.id;
    }
    public boolean getUpdated(){
        return this.updated;
    }
    public boolean getRemoved(){
        return this.removed;
    }
}
