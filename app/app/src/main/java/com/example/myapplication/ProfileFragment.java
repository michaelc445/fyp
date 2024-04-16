package com.example.myapplication;

import android.os.Bundle;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;
import android.widget.Toast;

import androidx.fragment.app.Fragment;

import com.example.myapplication.data.model.LoggedInUser;
import com.michaelc445.messages.PosterAppGrpc;
import com.michaelc445.messages.ProfileRequest;
import com.michaelc445.messages.ProfileResponse;

import java.util.concurrent.TimeUnit;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class ProfileFragment extends Fragment {


    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        // Inflate the layout for this fragment
        View view = inflater.inflate(R.layout.fragment_profile, container, false);

        TextView username = view.findViewById(R.id.profile_username);
        TextView party = view.findViewById(R.id.profile_party);
        TextView placedPosters = view.findViewById(R.id.placed_posters);
        TextView removedPosters = view.findViewById(R.id.removed_posters);

        try{
            LoggedInUser user = (LoggedInUser) getActivity().getIntent().getSerializableExtra("user");
            ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
            PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
            ProfileRequest req = ProfileRequest.newBuilder()
                    .setAuthKey(user.getAuthKey())
                    .setPartyId(user.getPartyId())
                    .setUserId(user.getUserId())
                    .build();
            ProfileResponse res = bStub.retrieveProfileStats(req);
            username.setText(username.getText()+user.getUserName());
            party.setText(party.getText()+user.getPartyName());
            placedPosters.setText(placedPosters.getText()+Integer.toString(res.getPlacedPosters()));
            removedPosters.setText(removedPosters.getText()+Integer.toString(res.getRemovedPosters()));
        }catch(Exception e){
            Toast.makeText(getContext(),"Failed to load profile data: "+e.getMessage(),Toast.LENGTH_LONG).show();
            Log.e("Profile","failed to load profile data",e);
        }


        return view;
    }
}
