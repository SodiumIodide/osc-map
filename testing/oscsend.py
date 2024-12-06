from pythonosc import udp_client

client = udp_client.SimpleUDPClient("10.1.10.77", 8005)
client.send_message("/cs/ping", "1")
