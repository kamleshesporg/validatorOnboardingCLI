
7. Initialize new node with same chain id
[[[[	ethermintd init node2name --chain-id os_9000-1 --home ~/.node2 folder-name
]]]]		
8. Copy genesis
[[[[	cp ~/.ethermintd/config/genesis.json ~/.node2 folder-name/config/
]]]]
9. Open config.toml and app.toml for new node
	Change Ports below should be different from node1 ports
App.toml -	
[grpc]
[grpc-web]
[json-rpc]
[api]

Config.toml -
[p2p]
laddr = "tcp://0.0.0.0:26666" 
external_address = "127.0.0.1:26666"
addr_book_strict = true → false
allow_duplicate_ip = false →  true
persistent_peers = "<first-node-id>@127.0.0.1:26656"  
[rpc]
laddr = "tcp://0.0.0.0:26667"

To get first node id use -
[[[[ethermintd tendermint show-node-id --home ~/.ethermintd
]]]]

10. Create second node address

[[[[ethermintd keys add node2name --algo eth_secp256k1 --keyring-backend test --home ~/.node2 folder-name
]]]]
11. Send funds to second node address
         [[[[NEW_VAL_ADDR=$(ethermintd keys show node2name -a --home ~/.node2 folder-name 
         --keyring-backend test)]]]]

[[[[ethermintd tx bank send validator $NEW_VAL_ADDR 5000000aphoton --chain-id os_9000-1 --keyring-backend test --node tcp://localhost:26657 --gas-prices 7aphoton --gas auto --gas-adjustment 1.1]]]]

Check trx status -
[[[[ethermintd query tx “hash” --node tcp://localhost:26657
]]]]
0 = success
Any other number = error


12. Start second node
[[[[
ethermintd start --home ~/.node2 folder-name --p2p.laddr tcp://0.0.0.0:26666 --rpc.laddr tcp://0.0.0.0:26667 --grpc.address 0.0.0.0:9092 --grpc-web.address 0.0.0.0:9093 --json-rpc.address 0.0.0.0:8547]]]]

Wait till blocks sync


13. Configure second node as validator
[[[[[ethermintd tx staking create-validator --amount=1000000aphoton --pubkey=$(ethermintd tendermint show-validator --home ~/.node2 folder-name) --moniker="node2name" --chain-id=os_9000-1 --commission-rate="0.10" --commission-max-rate="0.20" --commission-max-change-rate="0.01" --min-self-delegation="1" --from=node2name --keyring-backend=test --home ~/.node2 folder-name --node tcp://localhost:26667 --gas-prices 7aphoton --gas auto --gas-adjustment 1.1]]]]]

Check trx status -
[[[ethermintd query tx “hash” --node tcp://localhost:26657]]]