import os
from homeassistant_api import Client

URL = 'http://homeassistant.local:80'
TOKEN = os.getenv('HAKEY')

client = Client(URL, TOKEN)

entity_groups = client.get_entities()
print(entity_groups)
