package com.example.myapplication;

import android.app.DatePickerDialog;
import android.os.Bundle;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Button;
import android.widget.DatePicker;
import android.widget.EditText;
import android.widget.Toast;

import androidx.fragment.app.Fragment;

import com.example.myapplication.data.model.LoggedInUser;
import com.google.protobuf.Timestamp;
import com.michaelc445.messages.CreateElectionRequest;
import com.michaelc445.messages.CreateElectionResponse;
import com.michaelc445.messages.PosterAppGrpc;

import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.concurrent.TimeUnit;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

public class ElectionFragment extends Fragment {
    EditText start;
    EditText end;
    Button startDate, endDate, submit;
    Date electionStart, electionEnd;

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        // Inflate the layout for this fragment
        View view = inflater.inflate(R.layout.election_fragment, container, false);
        start = view.findViewById(R.id.start_date);
        start.setEnabled(false);
        end = view.findViewById(R.id.end_date);
        end.setEnabled(false);
        startDate = view.findViewById(R.id.editStartDate);
        endDate = view.findViewById(R.id.editEndDate);
        submit = view.findViewById(R.id.submit_date);
        startDate.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                DatePickerDialog d = new DatePickerDialog(getContext(), new DatePickerDialog.OnDateSetListener() {
                    @Override
                    public void onDateSet(DatePicker view, int year, int month, int dayOfMonth) {
                        electionStart = new Date(year-1900,month,dayOfMonth);
                        SimpleDateFormat dateFormat = new SimpleDateFormat("dd-MM-yyyy");
                        start.setText("Start date: "+dateFormat.format(electionStart));
                    }
                }, 2024, 0, 0);
                d.show();
            }
        });
        endDate.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                DatePickerDialog d = new DatePickerDialog(getContext(), new DatePickerDialog.OnDateSetListener() {
                    @Override
                    public void onDateSet(DatePicker view, int year, int month, int dayOfMonth) {
                        electionEnd = new Date(year-1900,month,dayOfMonth);
                        SimpleDateFormat dateFormat = new SimpleDateFormat("dd-MM-yyyy");
                        end.setText(String.format("End date: " + dateFormat.format(electionEnd)));
                    }
                }, 2024, 0, 0);
                d.show();
            }
        });
        submit.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                if (startDate == null || endDate == null){
                    Toast.makeText(getContext(),"start and end date must both be selected",Toast.LENGTH_LONG).show();
                    return;
                }
                try {
                    LoggedInUser user = UserData.getUser();
                    ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
                    PosterAppGrpc.PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
                    CreateElectionRequest req = CreateElectionRequest.newBuilder()
                            .setAuthKey(user.getAuthKey())
                            .setPartyId(user.getPartyId())
                            .setUserId(user.getUserId())
                            .setStartDate(Timestamp.newBuilder().setSeconds(electionStart.getTime()/1000).build())
                            .setElectionDate(Timestamp.newBuilder().setSeconds(electionEnd.getTime()/1000).build())
                            .build();
                    Log.e("election","start: "+req.getStartDate().getSeconds());
                    Log.e("election","election: "+req.getElectionDate().getSeconds());
                    CreateElectionResponse res = bStub.newElection(req);
                    Toast.makeText(getContext(),"election update successfully",Toast.LENGTH_LONG).show();
                }catch (Exception e){
                    Toast.makeText(getContext(),"failed to update election: "+e.getMessage(),Toast.LENGTH_LONG).show();
                    Log.e("election","failed to update election"+e.getMessage());
                }
            }
        });





        return view;
    }
}
