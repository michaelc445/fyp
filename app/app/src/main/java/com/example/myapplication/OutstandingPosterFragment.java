package com.example.myapplication;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Toast;

import androidx.fragment.app.Fragment;
import androidx.recyclerview.widget.LinearLayoutManager;
import androidx.recyclerview.widget.RecyclerView;

import com.example.myapplication.data.model.LoggedInUser;
import com.michaelc445.messages.PosterAppGrpc;
import com.michaelc445.messages.PosterTimeRequest;
import com.michaelc445.messages.PosterTimeResponse;
import com.michaelc445.messages.PosterUser;

import java.time.LocalDateTime;
import java.time.ZoneOffset;
import java.util.ArrayList;
import java.util.concurrent.TimeUnit;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class OutstandingPosterFragment extends Fragment {

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_outstanding_posters, container, false);

        ArrayList<PosterData> posters = getPosters();

        RecyclerView posterListRecyclerView = view.findViewById(R.id.outstanding_poster_recycler_view);
        PosterAdapter adapter = new PosterAdapter(posters);

        posterListRecyclerView.setLayoutManager(new LinearLayoutManager(getContext()));

        posterListRecyclerView.setAdapter(adapter);
        return view;
    }

    public ArrayList<PosterData> getPosters(){
        ArrayList<PosterData> posters = new ArrayList<>();
        try{
            LoggedInUser user = UserData.getUser();
            ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
            PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
            PosterTimeRequest req = PosterTimeRequest.newBuilder()
                    .setAuthKey(user.getAuthKey())
                    .setPartyId(user.getPartyId())
                    .setUserId(user.getUserId())
                    .build();
            PosterTimeResponse res = bStub.outstandingPosters(req);
            mChannel.shutdown().awaitTermination(2,TimeUnit.SECONDS);
            LocalDateTime t = LocalDateTime.ofEpochSecond(res.getRemovalDate().getSeconds(),res.getRemovalDate().getNanos(), ZoneOffset.UTC);
            for (PosterUser p : res.getPostersList()){
                posters.add(new PosterData(p.getPoster().getPosterid(),p.getUsername(),p.getFirstName(),p.getLastName(),t));
            }
            return posters;

        }catch(Exception e){
            Toast.makeText(getContext(),"failed to get outstanding posters: "+e.getMessage(),Toast.LENGTH_LONG).show();
        }

        return new ArrayList<>();
    }
}
