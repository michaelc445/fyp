package com.example.myapplication;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Button;
import android.widget.EditText;
import android.widget.Toast;

import androidx.fragment.app.Fragment;

import com.example.myapplication.data.model.LoggedInUser;
import com.michaelc445.messages.PosterAppGrpc;
import com.michaelc445.messages.RegisterPartyRequest;
import com.michaelc445.messages.RegisterPartyResponse;
import com.michaelc445.messages.ResponseCode;

import java.util.concurrent.TimeUnit;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class RegisterPartyFragment extends Fragment {


    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        // Inflate the layout for this fragment
        View view = inflater.inflate(R.layout.fragment_register_party, container, false);

        EditText userInputEditText = view.findViewById(R.id.party_name);
        Button submitButton = view.findViewById(R.id.submit_button);

        submitButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                // Get the user input
                String partyName = userInputEditText.getText().toString();
                try{
                    LoggedInUser user = UserData.getUser();
                    ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
                    PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
                    RegisterPartyRequest req = RegisterPartyRequest.newBuilder()
                            .setAuthKey(user.getAuthKey())
                            .setUserId(user.getUserId())
                            .setPartyName(partyName)
                            .build();
                    RegisterPartyResponse res = bStub.registerParty(req);
                    if (res.getCode() != ResponseCode.OK){
                        throw new Exception("failed to create party");
                    }
                    LoggedInUser newUser = new LoggedInUser(
                            user.getUserId(),
                            user.getUserName(),
                            res.getAuthKey(),
                            res.getPartyId(),
                            partyName
                    );
                    UserData.setUser(newUser);
                    Toast.makeText(getActivity(),"Party created successfully",Toast.LENGTH_LONG).show();

                }catch(Exception e){
                    Toast.makeText(getActivity(),"Failed to create party: "+e.getMessage(),Toast.LENGTH_LONG).show();
                }
            }
        });

        return view;
    }
}
