package com.example.myapplication;

import android.view.View;
import android.widget.Button;
import android.widget.TextView;

import androidx.recyclerview.widget.RecyclerView;

public class PartyViewHolder extends RecyclerView.ViewHolder {

    TextView partyNameTextView;
    Button joinButton;


    public PartyViewHolder(View itemView) {
        super(itemView);
        partyNameTextView = itemView.findViewById(R.id.party_name);
        joinButton = itemView.findViewById(R.id.join_party);
    }

}
