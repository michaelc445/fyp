package com.example.myapplication;

import android.view.View;
import android.widget.ImageButton;
import android.widget.TextView;

import androidx.recyclerview.widget.RecyclerView;

public class UserViewHolder extends RecyclerView.ViewHolder {

    TextView userNameTextView;
    ImageButton approveButton;
    ImageButton denyButton; // Consider using a single button with dynamic text

    public UserViewHolder(View itemView) {
        super(itemView);
        userNameTextView = itemView.findViewById(R.id.username);
        approveButton = itemView.findViewById(R.id.approve);
        denyButton = itemView.findViewById(R.id.deny);
    }

}
