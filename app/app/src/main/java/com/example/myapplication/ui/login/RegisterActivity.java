package com.example.myapplication.ui.login;

import android.os.Bundle;
import android.text.TextUtils;
import android.view.View;
import android.widget.Button;
import android.widget.EditText;
import android.widget.Toast;

import androidx.appcompat.app.AppCompatActivity;

import com.example.myapplication.Data;
import com.example.myapplication.R;
import com.michaelc445.messages.PosterAppGrpc;
import com.michaelc445.messages.PosterAppGrpc.PosterAppBlockingStub;
import com.michaelc445.messages.RegisterAccountRequest;
import com.michaelc445.messages.RegisterAccountResponse;
import com.michaelc445.messages.ResponseCode;

import java.util.concurrent.TimeUnit;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
public class RegisterActivity extends AppCompatActivity {

    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        setContentView(R.layout.activity_register);

        EditText firstNameEditText = findViewById(R.id.firstName);
        EditText lastNameEditText = findViewById(R.id.lastName);
        EditText usernameEditText = findViewById(R.id.username);
        EditText passwordEditText = findViewById(R.id.password);
        Button loginButton = findViewById(R.id.register);

        loginButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {

                if (TextUtils.isEmpty(firstNameEditText.getText())){
                    firstNameEditText.setError("First name is required");
                    return;
                }
                if (TextUtils.isEmpty(lastNameEditText.getText())){
                    lastNameEditText.setError("Last name is required");
                    return;
                }
                if (TextUtils.isEmpty(usernameEditText.getText())){
                    usernameEditText.setError("username is required");
                    return;
                }
                if (usernameEditText.getText().length() < 6){
                    usernameEditText.setError("username must be > 5 characters");
                    return;
                }
                if (TextUtils.isEmpty(passwordEditText.getText())){
                    passwordEditText.setError("password is required");
                    return;
                }
                if (usernameEditText.getText().length() < 6){
                    usernameEditText.setError("password must be > 5 characters");
                    return;
                }
                ManagedChannel mChannel = ManagedChannelBuilder.forAddress(Data.getAddress(),Data.getPort()).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();

                PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);

                RegisterAccountRequest req = RegisterAccountRequest.newBuilder()
                        .setFirstName(firstNameEditText.getText().toString())
                        .setLastName(lastNameEditText.getText().toString())
                        .setUsername(usernameEditText.getText().toString())
                        .setPassword(passwordEditText.getText().toString()).build();

                try {
                    RegisterAccountResponse res = bStub.registerAccount(req);
                    if (res.getCode() == ResponseCode.FAILED){
                        Toast.makeText(getApplicationContext(), "Failed to register account",Toast.LENGTH_SHORT).show();
                        return;
                    }

                    Toast.makeText(getApplicationContext(), "Account registered successfully",Toast.LENGTH_SHORT).show();
                    finish();
                } catch (Exception e){
                    Toast.makeText(getApplicationContext(), "Failed to register account",Toast.LENGTH_SHORT).show();
                }
                mChannel.shutdown();
            }
        });
    }

}