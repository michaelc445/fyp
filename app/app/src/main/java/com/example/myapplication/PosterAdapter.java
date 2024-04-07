package com.example.myapplication;

import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;

import androidx.annotation.NonNull;
import androidx.recyclerview.widget.RecyclerView;

import java.time.Duration;
import java.time.LocalDateTime;
import java.util.List;


public class PosterAdapter extends RecyclerView.Adapter<PosterViewHolder>{
    private final List<PosterData> posters;

    public PosterAdapter(List<PosterData> posters){

        this.posters = posters;
    }
    @Override
    public PosterViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View view = LayoutInflater.from(parent.getContext()).inflate(R.layout.poster_item, parent, false);
        return new PosterViewHolder(view);
    }
    @Override
    public void onBindViewHolder(@NonNull PosterViewHolder holder, int position) {
        PosterData poster = posters.get(position);
        holder.posterIdTextView.setText("Poster ID: "+poster.getPosterId());
        holder.usernameTextView.setText("Username: "+poster.getUsername());
        holder.firstLastTextView.setText("Full Name: "+poster.getFirstName()+" "+poster.getLastName());
        holder.timeToRemovalTextView.setText("Remaining time: "+(Duration.between(LocalDateTime.now(),poster.getRemovalDate()).toDays()+7)+" days");

    }

    @Override
    public int getItemCount() {
        return this.posters.size();
    }
}
