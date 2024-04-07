package com.example.myapplication;

import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Toast;

import androidx.annotation.NonNull;
import androidx.recyclerview.widget.RecyclerView;

import com.example.myapplication.data.model.LoggedInUser;
import com.michaelc445.messages.JoinPartyRequest;
import com.michaelc445.messages.JoinPartyResponse;
import com.michaelc445.messages.PosterAppGrpc;

import java.util.List;
import java.util.concurrent.TimeUnit;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class PartyAdapter extends RecyclerView.Adapter<PartyViewHolder>{

    private List<Party> parties;

    public PartyAdapter(List<Party> parties){
        this.parties = parties;
    }
    @Override
    public PartyViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View view = LayoutInflater.from(parent.getContext()).inflate(R.layout.party_list_item, parent, false);
        return new PartyViewHolder(view);
    }

    @Override
    public void onBindViewHolder(@NonNull PartyViewHolder holder, int position) {
        Party party = parties.get(position);
        holder.partyNameTextView.setText(party.getPartyName());
        holder.joinButton.setOnClickListener(v -> {
            try{
                LoggedInUser localUser = UserData.getUser();
                ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
                PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
                JoinPartyRequest req = JoinPartyRequest.newBuilder()
                        .setUserId(localUser.getUserId())
                        .setAuthKey(localUser.getAuthKey())
                        .setPartyId(party.getPartyID())
                        .build();
                JoinPartyResponse res = bStub.joinParty(req);
                mChannel.shutdownNow().awaitTermination(2,TimeUnit.SECONDS);
                Toast.makeText(v.getContext(), "Join request sent. Please wait for response.",Toast.LENGTH_LONG).show();
            }catch (Exception e){
                Toast.makeText(v.getContext(),"Failed to join party: "+e.getMessage(),Toast.LENGTH_LONG).show();
                Log.d("Join party", "failed to join party: "+e.getMessage());
            }
        });
    }

    @Override
    public int getItemCount() {
        return this.parties.size();
    }
}
