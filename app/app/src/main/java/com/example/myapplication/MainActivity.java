package com.example.myapplication;

import android.content.Intent;
import android.os.Bundle;
import android.view.View;
import android.widget.Button;

import androidx.appcompat.app.AppCompatActivity;

import com.example.myapplication.ui.login.LoginActivity;
import com.example.myapplication.ui.login.RegisterActivity;

public class MainActivity extends AppCompatActivity {

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        Button loginButton = findViewById(R.id.loginButton);
        loginButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                startActivity(new Intent(MainActivity.this, LoginActivity.class));
            }
        });
        Button registerButton = findViewById(R.id.registerButton);
        registerButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                startActivity(new Intent(MainActivity.this, RegisterActivity.class));
            }
        });

//        ManagedChannel mChannel = ManagedChannelBuilder.forAddress("192.168.0.194",50051).usePlaintext().enableRetry().keepAliveTime(10, TimeUnit.SECONDS).build();
//
//        PosterAppBlockingStub bStub = PosterAppGrpc.newBlockingStub(mChannel);
//
//        RegisterAccountRequest r = RegisterAccountRequest.newBuilder()
//                .setFirstName("Michael")
//                .setLastName("john")
//                .setUsername("hahaha")
//                .setPassword("lolol").build();
//        RegisterAccountResponse res = bStub.registerAccount(r);
//        System.out.println(res);
    }







}