from pythonosc import dispatcher, osc_server

# This function will print out any OSC message received
def print_osc_command(osc_address, *args):
    print(f"Received OSC message: {osc_address} with arguments: {args}")

# Set up the dispatcher to handle all OSC messages
dispatcher = dispatcher.Dispatcher()
dispatcher.set_default_handler(print_osc_command)  # This handles any OSC address

# Define the IP and port your OSC listener is using
ip = "10.1.10.203"  # Listen on all network interfaces
port = 8006      # Ensure this matches the port set on ColorSource AV

# Create the OSC server to listen for incoming messages
server = osc_server.ThreadingOSCUDPServer((ip, port), dispatcher)
print(f"Listening for OSC messages on {ip}:{port}")
server.serve_forever()
