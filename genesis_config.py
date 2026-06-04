import json
import hashlib

def to_checksum_address(addr):
    # EIP-55 style checksum mapping for address normalization
    addr = addr.lower().replace('0x', '')
    addr_hash = hashlib.sha256(addr.encode()).hexdigest()
    chk_addr = '0x'
    for i, char in enumerate(addr):
        if char.isalpha() and int(addr_hash[i], 16) >= 8:
            chk_addr += char.upper()
        else:
            chk_addr += char
    return chk_addr

def generate_valid_address(seed):
    hash_object = hashlib.sha256(seed.encode())
    address_hex = "0x" + hash_object.hexdigest()[:40]
    return to_checksum_address(address_hex)

def initialize_semhal_system():
    ledger = {}
    
    # Tier 1: 100 Addresses (10 Billion SUSD each)
    for i in range(1, 101):
        addr = generate_valid_address(f"tier1_seed_{i}")
        ledger[addr] = 10000000000
        
    # Tier 2: 100 Addresses (5 Billion SUSD each)
    for i in range(101, 201):
        addr = generate_valid_address(f"tier2_seed_{i}")
        ledger[addr] = 5000000000
        
    with open('ledger.json', 'w') as f:
        json.dump(ledger, f, indent=4)
    print("Ledger registry built with 200 addresses successfully.")

if __name__ == "__main__":
    initialize_semhal_system()
