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
import com.michaelc445.messages.Member;
import com.michaelc445.messages.PosterAppGrpc;
import com.michaelc445.messages.RetrieveJoinRequest;
import com.michaelc445.messages.RetrieveJoinResponse;

import java.util.ArrayList;
import java.util.concurrent.TimeUnit;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class ApproveMembersFragment extends Fragment {


    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_approve_members, container, false);
        ArrayList<User> users = getMemberRequests();

        RecyclerView userListRecyclerView = view.findViewById(R.id.user_list_recycler_view);
        UserAdapter adapter = new UserAdapter(users);

        userListRecyclerView.setLayoutManager(new LinearLayoutManager(getContext()));

        userListRecyclerView.setAdapter(adapter);

        return view;

    }
    public ArrayList<User> getMemberRequests(){
        try{
            LoggedInUser user = UserData.getUser();
            ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
            PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
            RetrieveJoinRequest req = RetrieveJoinRequest.newBuilder()
                    .setAuthKey(user.getAuthKey())
                    .setPartyId(user.getPartyId())
                    .setUserId(user.getUserId())
                    .build();
            RetrieveJoinResponse res = bStub.retrieveJoinRequests(req);
            mChannel.shutdown().awaitTermination(2,TimeUnit.SECONDS);
            ArrayList<User> users = new ArrayList<>();
            for (Member m : res.getMembersList()){
                users.add(new User(m.getUserId(),m.getFirstName(),m.getLastName()));
            }

            return users;

        }catch(Exception e){
            Toast.makeText(getActivity(),"failed to get join requests "+e.getMessage(),Toast.LENGTH_LONG).show();
        }

        return new ArrayList<>();
    }
}
