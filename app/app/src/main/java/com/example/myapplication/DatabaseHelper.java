package com.example.myapplication;

import android.annotation.SuppressLint;
import android.content.ContentValues;
import android.content.Context;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.database.sqlite.SQLiteOpenHelper;
import android.util.Log;

import org.osmdroid.util.GeoPoint;

import java.util.ArrayList;
import java.util.List;

public class DatabaseHelper extends SQLiteOpenHelper {

    private static final String DATABASE_NAME = "poster_database.db";
    private static final int DATABASE_VERSION = 1;

    public DatabaseHelper(Context context) {
        super(context, DATABASE_NAME, null, DATABASE_VERSION);
    }

    @Override
    public void onCreate(SQLiteDatabase db) {
        // id will be key for local database, will need to store server id seperately incase placing poster fails
        db.execSQL("CREATE TABLE IF NOT EXISTS PosterTable (id INTEGER PRIMARY KEY AUTOINCREMENT, posterID INTEGER UNIQUE, lat REAL, lng REAL, removed BOOLEAN, updateSent BOOLEAN)");

    }
    public List<Poster> getCachedPosters() {
        SQLiteDatabase db = this.getWritableDatabase();
        ArrayList<Poster> posters = new ArrayList<>();
        Cursor cursor = db.query("PosterTable", new String[]{"id","lat","lng","removed"},"updateSent = false",null,null,null,null);
        for (cursor.moveToFirst(); !cursor.isAfterLast(); cursor.moveToNext()) {
            @SuppressLint("Range") double longitude = cursor.getDouble(cursor.getColumnIndex("lng"));
            @SuppressLint("Range") double latitude = cursor.getDouble(cursor.getColumnIndex("lat"));
            @SuppressLint("Range") int localID = cursor.getInt(cursor.getColumnIndex("id"));
            @SuppressLint("Range") int removed = cursor.getInt(cursor.getColumnIndex("removed"));
            posters.add(new Poster(latitude,longitude,localID,false,removed==1));
        }
        cursor.close();
        return posters;
    }
    @Override
    public void onUpgrade(SQLiteDatabase db, int oldVersion, int newVersion) {

    }
    public GeoPoint removePoster(int posterId){
        SQLiteDatabase db = this.getWritableDatabase();
        ContentValues values = new ContentValues();
        db.beginTransaction();
        values.put("removed",true);
        int count = db.update("PosterTable",values,"posterID = "+posterId,null);
        if (count != 1){

            db.endTransaction();
            Log.d("Database helper", "failed to remove poster: "+posterId+" found "+count+" rows");
            return null;
        }

        Cursor cursor = db.query("PosterTable",new String[]{"lat","lng"},"posterID = "+posterId,null,null,null,null);
        try {
            cursor.moveToNext();
            @SuppressLint("Range") double longitude = cursor.getDouble(cursor.getColumnIndex("lng"));
            @SuppressLint("Range") double latitude = cursor.getDouble(cursor.getColumnIndex("lat"));
            db.setTransactionSuccessful();
            db.endTransaction();
            cursor.close();
            return new GeoPoint(latitude,longitude);
        }catch(Exception e){
            cursor.close();
            db.endTransaction();
            return null;
        }

    }
    public GeoPoint getLocation(int posterId){
        SQLiteDatabase db = this.getReadableDatabase();
        Cursor cursor = db.query("PosterTable",new String[]{"lat","lng"},"posterID = "+posterId,null,null,null,null);
        try {
            cursor.moveToNext();
            @SuppressLint("Range") double longitude = cursor.getDouble(cursor.getColumnIndex("lng"));
            @SuppressLint("Range") double latitude = cursor.getDouble(cursor.getColumnIndex("lat"));
            cursor.close();
            return new GeoPoint(latitude,longitude);
        }catch(Exception e){
            cursor.close();
            return null;
        }
    }
    public void updateDB(List<Poster> posters){
        SQLiteDatabase db = this.getWritableDatabase();
        db.beginTransaction();
        try {
            for (Poster poster: posters){
                String query = "INSERT OR REPLACE INTO PosterTable (posterID, lat, lng, removed, updateSent) " +
                        "VALUES (" + poster.getId() + ", " + poster.getLatitude() + ", " + poster.getLongitude() + ", " + poster.getRemoved() + ", "+poster.getUpdated()+")";

                db.execSQL(query);

            }
            db.setTransactionSuccessful();
        }catch(Exception e){
            Log.e("Database Helper","Failed to update posters",e);
        }finally {
            db.endTransaction();

        }

    }
    public ArrayList<Poster> getPosters(){
        SQLiteDatabase db = this.getReadableDatabase();
        Cursor cursor = db.query("PosterTable",new String[]{"lat","lng","posterID","updateSent"},"removed = false",null,null,null,null);
        ArrayList<Poster> posters = new ArrayList<>();

        for (cursor.moveToFirst(); !cursor.isAfterLast(); cursor.moveToNext()) {
            @SuppressLint("Range") double longitude = cursor.getDouble(cursor.getColumnIndex("lng"));
            @SuppressLint("Range") double latitude = cursor.getDouble(cursor.getColumnIndex("lat"));
            @SuppressLint("Range") int posterID = cursor.getInt(cursor.getColumnIndex("posterID"));
            @SuppressLint("Range") int updated = cursor.getInt(cursor.getColumnIndex("updateSent"));
            Poster p = new Poster(latitude,longitude,posterID,false,updated==1);
            posters.add(p);
        }

        cursor.close();
        return posters;
    }
    public void updatePoster(int localID, int newServerID){
        SQLiteDatabase db = this.getReadableDatabase();
        ContentValues values = new ContentValues();
        values.put("posterID",newServerID);
        values.put("updateSent",true);

        db.update("PosterTable",values,"id = "+localID,null);
    }
    public void insertPoster(Poster poster){
        SQLiteDatabase db = this.getWritableDatabase();
        ContentValues values = new ContentValues();
        if (poster.getIdInteger() != null) {
            values.put("posterID", poster.getIdInteger().intValue());
        }
        values.put("lat",poster.getLatitude());
        values.put("lng",poster.getLongitude());
        values.put("removed",false);
        values.put("updateSent",poster.getUpdated());

        db.insert("PosterTable",null,values);

    }
    public void removePoster(Poster poster){
        SQLiteDatabase db = this.getWritableDatabase();

        db.delete("PosterTable","posterID = "+poster.getId(), null);

    }
}
