package com.example.myapplication;

import static android.content.Context.LOCATION_SERVICE;

import android.Manifest;
import android.content.Context;
import android.content.SharedPreferences;
import android.content.pm.PackageManager;
import android.location.LocationManager;
import android.net.ConnectivityManager;
import android.net.NetworkInfo;
import android.os.Bundle;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Button;
import android.widget.Toast;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.core.app.ActivityCompat;
import androidx.fragment.app.Fragment;

import com.example.myapplication.data.model.LoggedInUser;
import com.google.android.material.floatingactionbutton.FloatingActionButton;
import com.google.protobuf.Timestamp;
import com.michaelc445.messages.Location;
import com.michaelc445.messages.PlacementRequest;
import com.michaelc445.messages.PlacementResponse;
import com.michaelc445.messages.PosterAppGrpc;
import com.michaelc445.messages.RemovePosterRequest;
import com.michaelc445.messages.RemovePosterResponse;
import com.michaelc445.messages.UpdateRequest;
import com.michaelc445.messages.UpdateResponse;

import org.osmdroid.config.Configuration;
import org.osmdroid.library.BuildConfig;
import org.osmdroid.tileprovider.tilesource.TileSourceFactory;
import org.osmdroid.util.GeoPoint;
import org.osmdroid.views.MapView;
import org.osmdroid.views.overlay.Marker;
import org.osmdroid.views.overlay.Overlay;
import org.osmdroid.views.overlay.OverlayManager;

import java.util.ArrayList;
import java.util.List;
import java.util.ListIterator;
import java.util.concurrent.TimeUnit;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class MapFragmentRemovePoster extends Fragment {
    private static final int LOCATION_PERMISSION_REQUEST_CODE = 1;
    private MapView mapView;
    private LocationManager locationManager;
    private android.location.Location lastKnownLocation;

//    private Marker userMarker;


    @Nullable
    @Override
    public View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container, @Nullable Bundle savedInstanceState) {
        View rootView = inflater.inflate(R.layout.fragment_map, container, false);

        if(isNetworkAvailable()) {
            sendCachedUpdates(UserData.getUser());
        }
        // Initialize the map view
        mapView = rootView.findViewById(R.id.map);
        mapView.setTileSource(TileSourceFactory.MAPNIK);

        // Enable pinch zooming
        mapView.setMultiTouchControls(true);

        DatabaseHelper db = new DatabaseHelper(getContext());
        ArrayList<Poster> posters = db.getPosters();

        for (Poster p :posters){
            GeoPoint geoL = new GeoPoint(p.getLatitude(), p.getLongitude());
            // Add a marker at the user's location
            Marker marker = new Marker(mapView);

            marker.setPosition(geoL);
            marker.setIcon(getResources().getDrawable(org.osmdroid.library.R.drawable.marker_default)); // Set a custom marker icon
            mapView.getOverlays().add(marker);
        }

        Button removePosterButton = rootView.findViewById(R.id.poster_button);
        removePosterButton.setText("Remove Poster");

        removePosterButton.setOnClickListener(v -> {
            android.location.Location location = getLocation();
            LoggedInUser  user = UserData.getUser();
            ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
            PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
            RemovePosterRequest req = RemovePosterRequest.newBuilder()
                    .setAuthKey(user.getAuthKey())
                    .setLocation(Location.newBuilder().setLng(location.getLongitude()).setLat(location.getLatitude()).build())
                    .setUserId(user.getUserId())
                    .setPartyId(user.getPartyId()).build();

            try {
                // remove the poster
                RemovePosterResponse res = bStub.removePoster(req);

                GeoPoint p = db.removePoster(res.getPosterid());
                removePosterFromMap(p);
            }catch (Exception e){
                // save poster offline and send poster later
                Log.d("remove poster",""+e);
                Toast.makeText(getContext(),"Failed to remove poster",Toast.LENGTH_SHORT).show();
//                db.insertPoster(new Poster(location.getLatitude(),location.getLongitude(),null,false,true));
            }
            mChannel.shutdown();
        });

        FloatingActionButton updateButton = rootView.findViewById(R.id.refresh_button);
        updateButton.setOnClickListener(v -> {
            getUpdates(UserData.getUser());
            mapView.getOverlay().clear();
            ArrayList<Poster> posterList = db.getPosters();

            for (Poster p :posterList){
                GeoPoint geoL = new GeoPoint(p.getLatitude(), p.getLongitude());
                // Add a marker at the user's location
                Marker marker = new Marker(mapView);

                marker.setPosition(geoL);
                marker.setIcon(getResources().getDrawable(org.osmdroid.library.R.drawable.marker_default)); // Set a custom marker icon
                mapView.getOverlays().add(marker);
            }
            mapView.invalidate();
        });
        // Configure the map view
        Configuration.getInstance().setUserAgentValue(BuildConfig.APPLICATION_ID);

        getUpdates(UserData.getUser());
        // Initialize fused location client
        locationManager = (LocationManager) requireContext().getSystemService(Context.LOCATION_SERVICE);

        // Check and request location permissions if not granted
        if (ActivityCompat.checkSelfPermission(requireContext(), Manifest.permission.ACCESS_FINE_LOCATION) != PackageManager.PERMISSION_GRANTED) {
            ActivityCompat.requestPermissions(requireActivity(), new String[]{Manifest.permission.ACCESS_FINE_LOCATION}, LOCATION_PERMISSION_REQUEST_CODE);
        } else {
            // Permissions already granted, update map with user's location
            updateMapWithLocation();
//            locationManager.requestLocationUpdates(LocationManager.GPS_PROVIDER, 30000, 10, (location) -> {
//                // Update map marker with new location
//                updateMapMarker(location);
//            });
        }

        return rootView;
    }
    public void removePosterFromMap(GeoPoint p){
        if (p == null){
            Log.d("Remove fragment", "location p is null");
            return;
        }
        // Get the OverlayManager from the mapView
        OverlayManager overlayManager = mapView.getOverlayManager();

        // Get all markers
        ListIterator<Overlay> markers = overlayManager.listIterator();
        for (ListIterator<Overlay> it = markers; it.hasNext(); ) {
            Overlay m = it.next();
            if (m instanceof  Marker){
                Marker t = (Marker) m;
                GeoPoint markerPosition = t.getPosition();
                if (markerPosition == null){
                    continue;
                }
                if (markerPosition.getLatitude() == p.getLatitude() && markerPosition.getLongitude() == p.getLongitude()){
                    overlayManager.remove(m);
                    mapView.invalidate();
                    return;
                }
            }


        }
    }
    private boolean isNetworkAvailable() {
        ConnectivityManager connectivityManager
                = (ConnectivityManager) getContext().getSystemService(Context.CONNECTIVITY_SERVICE);
        NetworkInfo activeNetworkInfo = connectivityManager != null ? connectivityManager.getActiveNetworkInfo() : null;
        return activeNetworkInfo != null && activeNetworkInfo.isConnected();
    }
    public void sendCachedUpdates(LoggedInUser user){
        DatabaseHelper db = new DatabaseHelper(getContext());
        List<Poster> posters = db.getCachedPosters();
        if (posters.size()==0){
            return;
        }
        ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
        PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
        for (Poster p: posters){
            try{
                if (p.getRemoved()){
                    RemovePosterRequest req = RemovePosterRequest.newBuilder()
                            .setAuthKey(user.getAuthKey())
                            .setUserId(user.getUserId())
                            .setPartyId(user.getPartyId())
                            .setLocation(Location.newBuilder()
                                    .setLat(p.getLatitude())
                                    .setLng(p.getLongitude()).build()
                            ).build();
                    RemovePosterResponse res = bStub.removePoster(req);
                    db.updatePoster(p.getId(),res.getPosterid());
                }else{
                    PlacementRequest req = PlacementRequest.newBuilder()
                            .setAuthKey(user.getAuthKey())
                            .setUserId(user.getUserId())
                            .setPartyId(user.getPartyId())
                            .setLocation(Location.newBuilder()
                                    .setLat(p.getLatitude())
                                    .setLng(p.getLongitude()).build()
                            ).build();

                    PlacementResponse res = bStub.placePoster(req);
                    db.updatePoster(p.getId(),res.getPosterId());
                }

            } catch(Exception e){
                Log.e("Sending cached updates","Update failed: "+e);
            }
        }
    }
    @Override
    public void onViewCreated(View view, Bundle savedInstanceState){
        super.onViewCreated(view, savedInstanceState);
        if (ActivityCompat.checkSelfPermission(requireContext(), Manifest.permission.ACCESS_FINE_LOCATION) != PackageManager.PERMISSION_GRANTED) {
            ActivityCompat.requestPermissions(requireActivity(), new String[]{Manifest.permission.ACCESS_FINE_LOCATION}, LOCATION_PERMISSION_REQUEST_CODE);
        } else {
            // Permissions already granted, update map with user's location
//            locationManager.requestLocationUpdates(LocationManager.GPS_PROVIDER, 30000, 10, (location) -> {
//                // Update map marker with new location
//                updateMapMarker(location);
//            });
        }

    }
    public void getUpdates(LoggedInUser user){

        ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
        PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
        SharedPreferences preferences = getContext().getSharedPreferences("last_updated",Context.MODE_PRIVATE);

        Long lastUpdated = preferences.getLong("last_updated",0L);

        try {
            // remove the poster
            UpdateRequest req = UpdateRequest.newBuilder()
                    .setAuthKey(user.getAuthKey())
                    .setUserId(user.getUserId())
                    .setPartyid(user.getPartyId())
                    .setLastUpdated(Timestamp.newBuilder().setSeconds(lastUpdated/1000).build())
                    .build();
            UpdateResponse res = bStub.retrieveUpdates(req);
            ArrayList<Poster> posterList = new ArrayList<>();
            for (com.michaelc445.messages.Poster p : res.getPostersList()){
                Poster newPoster = new Poster(p.getLocation().getLat(),
                        p.getLocation().getLng(),p.getPosterid(),true,p.getRemoved());
                posterList.add(newPoster);
            }
            DatabaseHelper db = new DatabaseHelper(getContext());
            db.updateDB(posterList);
            preferences.edit().putLong("last_updated",System.currentTimeMillis()).apply();


        }catch (Exception e){
            // save poster offline and send poster later
            Toast.makeText(getContext(),"Failed to get updates from server",Toast.LENGTH_SHORT).show();
        }
        mChannel.shutdown();
    }

    @Override
    public void onResume() {
        super.onResume();
        mapView.onResume();
    }

    @Override
    public void onPause() {
        super.onPause();
        mapView.onPause();
    }
    // solution credit to: https://stackoverflow.com/a/20465781
    private android.location.Location getLocation(){
        android.location.Location bestLocation = null;
        if (ActivityCompat.checkSelfPermission(requireContext(), Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED) {

            LocationManager LocationManager = (LocationManager)getContext().getSystemService(LOCATION_SERVICE);
            List<String> providers = LocationManager.getProviders(true);

            for (String provider : providers) {
                android.location.Location l = LocationManager.getLastKnownLocation(provider);
                if (l == null) {
                    continue;
                }
                if (bestLocation == null || l.getAccuracy() < bestLocation.getAccuracy()) {
                    // Found best last known location: %s", l);
                    bestLocation = l;
                }
            }
        }
        return bestLocation;
    }
    private void updateMapWithLocation() {
        // Get last known location
        android.location.Location l = getLocation();
        if (l != null && mapView != null){
            GeoPoint geoL = new GeoPoint(l.getLatitude(), l.getLongitude());
            mapView.getController().animateTo(geoL);
            mapView.getController().setCenter(geoL);
            mapView.getController().setZoom(18.0); // Set a more appropriate zoom level
            mapView.invalidate();
        }
    }

//    private void updateMapMarker(android.location.Location location) {
//        if (location != null && mapView != null) {
//            GeoPoint userLocation = new GeoPoint(location.getLatitude(), location.getLongitude());
//            if (userMarker != null) {
//                // Update existing marker position
//                userMarker.setPosition(userLocation);
//                mapView.invalidate(); // Trigger a redraw
//            } else {
//                // Create and add a new marker for user location
//                userMarker = new Marker(mapView);
//                userMarker.setPosition(userLocation);
//                // Set a custom marker icon or other properties as needed
//                userMarker.setIcon(getResources().getDrawable(org.osmdroid.library.R.drawable.marker_default));
//                mapView.getOverlays().add(userMarker);
//                mapView.invalidate();
//
//            }
//        }
//    }
}