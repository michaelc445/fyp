package com.example.myapplication;

import android.view.View;
import android.widget.TextView;

import androidx.recyclerview.widget.RecyclerView;

public class PosterViewHolder extends RecyclerView.ViewHolder {

    TextView posterIdTextView;
    TextView usernameTextView;
    TextView firstLastTextView;
    TextView timeToRemovalTextView;



    public PosterViewHolder(View itemView) {
        super(itemView);
        posterIdTextView = itemView.findViewById(R.id.poster_id);
        usernameTextView = itemView.findViewById(R.id.username_posters);
        firstLastTextView = itemView.findViewById(R.id.first_last_name);
        timeToRemovalTextView = itemView.findViewById(R.id.time_to_removal);

    }
}
