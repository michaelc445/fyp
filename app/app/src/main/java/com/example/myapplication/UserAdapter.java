package com.example.myapplication;

import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Toast;

import androidx.recyclerview.widget.RecyclerView;

import com.example.myapplication.data.model.LoggedInUser;
import com.michaelc445.messages.ApproveMemberRequest;
import com.michaelc445.messages.ApproveMemberResponse;
import com.michaelc445.messages.Member;
import com.michaelc445.messages.PosterAppGrpc;

import java.util.List;
import java.util.concurrent.TimeUnit;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class UserAdapter extends RecyclerView.Adapter<UserViewHolder> {
    private List<User> users;

    public UserAdapter(List<User> users) {
        this.users = users;
    }

    @Override
    public UserViewHolder onCreateViewHolder(ViewGroup parent, int viewType) {
        // Inflate the layout for a single user item
        View view = LayoutInflater.from(parent.getContext()).inflate(R.layout.user_list_item, parent, false);
        return new UserViewHolder(view);
    }

    public void onBindViewHolder(UserViewHolder holder, int position) {
        User user = users.get(position);
        holder.userNameTextView.setText(user.getFirstName()+" "+user.getLastName()+" "+user.getUserID());
        holder.approveButton.setOnClickListener(v -> {
            User t = users.get(holder.getAdapterPosition());
            try{
                LoggedInUser localUser = UserData.getUser();
                ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
                PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
                ApproveMemberRequest req = ApproveMemberRequest.newBuilder()
                        .setUserId(localUser.getUserId())
                        .setAuthKey(localUser.getAuthKey())
                        .setPartyId(localUser.getPartyId())
                        .addApprovedMembers(
                                Member.newBuilder()
                                        .setUserId(t.getUserID())
                                        .setFirstName(t.getFirstName())
                                        .setLastName(t.getLastName())
                                        .build()
                        )
                        .build();
                ApproveMemberResponse res = bStub.approveMembers(req);
                mChannel.shutdownNow().awaitTermination(2,TimeUnit.SECONDS);
                removeItemFromList(holder.getAdapterPosition());
            }catch(Exception e){
                Toast.makeText(
                        v.getContext(),
                        "Failed to approve member: "+user.getFirstName()+" "+user.getLastName()+" "+e.getMessage(),
                        Toast.LENGTH_SHORT
                        )
                        .show();
                Log.d("approve",""+e);
            }
        });

        holder.denyButton.setOnClickListener(v -> {
            try{

                User t = users.get(holder.getAdapterPosition());
                LoggedInUser localUser = UserData.getUser();
                ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
                PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
                ApproveMemberRequest req = ApproveMemberRequest.newBuilder()
                        .setUserId(localUser.getUserId())
                        .setAuthKey(localUser.getAuthKey())
                        .setPartyId(localUser.getPartyId())
                        .addDeniedMembers(
                                Member.newBuilder()
                                        .setUserId(t.getUserID())
                                        .setFirstName(t.getFirstName())
                                        .setLastName(t.getLastName())
                                        .build()
                        )
                        .build();
                ApproveMemberResponse res = bStub.approveMembers(req);
                mChannel.shutdownNow().awaitTermination(2,TimeUnit.SECONDS);
                removeItemFromList(holder.getAdapterPosition());
            }catch(Exception e){
                Toast.makeText(
                                v.getContext(),
                                "Failed to deny member: "+user.getFirstName()+" "+user.getLastName()+" "+e.getMessage(),
                                Toast.LENGTH_SHORT
                        )
                        .show();
                Log.d("deny",""+e);
            }
        });
    }

    public void removeItemFromList(int position){
        if (position >=0 && position < users.size()){
            users.remove(position);
            notifyItemRemoved(position);
        }
    }
    @Override
    public int getItemCount() {
        return users.size();
    }
}

