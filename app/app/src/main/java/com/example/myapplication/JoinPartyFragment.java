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
import com.michaelc445.messages.RetrievePartiesRequest;
import com.michaelc445.messages.RetrievePartiesResponse;

import java.util.ArrayList;
import java.util.concurrent.TimeUnit;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class JoinPartyFragment extends Fragment {

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_join_party, container, false);

        ArrayList<Party> parties = getParties();

        RecyclerView partyListRecyclerView = view.findViewById(R.id.party_list_recycler_view);
        PartyAdapter adapter = new PartyAdapter(parties);

        partyListRecyclerView.setLayoutManager(new LinearLayoutManager(getContext()));

        partyListRecyclerView.setAdapter(adapter);
        return view;
    }

    public ArrayList<Party> getParties(){

        try{
            LoggedInUser user = UserData.getUser();
            ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
            PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
            RetrievePartiesRequest req = RetrievePartiesRequest.newBuilder()
                    .setAuthKey(user.getAuthKey())
                    .build();
            RetrievePartiesResponse res = bStub.retrieveParties(req);
            mChannel.shutdown().awaitTermination(2,TimeUnit.SECONDS);
            ArrayList<Party> parties = new ArrayList<>();
            for (com.michaelc445.messages.Party m : res.getPartiesList()){
                parties.add(new Party(m.getName(),m.getPartyID()));
            }

            return parties;

        }catch(Exception e){
            Toast.makeText(getActivity(),"failed to get party list "+e.getMessage(),Toast.LENGTH_LONG).show();
        }

        return new ArrayList<>();
    }
}
