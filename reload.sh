# 2.1 means "1st pane of the 2nd window"
tmux send-keys -t 2.1 C-c
sleep 0.5
tmux send-keys -t 2.1 clear C-m
sleep 0.2
tmux send-keys -t 2.1 "go run ./cmd/client" C-m
sleep 0.2
