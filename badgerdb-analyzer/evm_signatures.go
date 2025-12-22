package main

// EVMEventSignatures maps Keccak256 event signature hashes to human-readable signatures.
// The hash is already computed and stored in topics[0] by the EVM - we just look it up.
// Hashes are lowercase hex strings without "0x" prefix.
//
// This is a curated list focused on protocols deployed on Oasis Network (Sapphire and Emerald).
// For comprehensive coverage, consider integrating with external signature databases like
// 4byte.directory or Etherscan. Users can extend this map with project-specific signatures.
var EVMEventSignatures = map[string]string{
	// ERC-20 Token Standard (Used by all tokens on Oasis)
	"ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef": "Transfer(address,address,uint256)",
	"8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925": "Approval(address,address,uint256)",

	// ERC-721 NFT Standard
	"17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31": "ApprovalForAll(address,address,bool)",

	// ERC-1155 Multi-Token Standard
	"c3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62": "TransferSingle(address,address,address,uint256,uint256)",
	"4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb": "TransferBatch(address,address,address,uint256[],uint256[])",

	// Ownable / Access Control (Common pattern in Oasis contracts)
	"8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0": "OwnershipTransferred(address,address)",
	"2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d": "RoleGranted(bytes32,address,address)",
	"f6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b": "RoleRevoked(bytes32,address,address)",

	// Pausable (Common security pattern)
	"62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258": "Paused(address)",
	"5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa": "Unpaused(address)",

	// Wrapped ETH/ROSE (wROSE: 0x8Bc2B030b299964eEfb5e1e0b36991352E56D2D3)
	"e1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c": "Deposit(address,uint256)",
	"7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65": "Withdrawal(address,uint256)",

	// Uniswap V2 / DEX Events (YuzuSwap, ValleySwap, LilacSwap on Emerald)
	"1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1": "Sync(uint112,uint112)",
	"d78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822": "Swap(address,uint256,uint256,uint256,uint256,address)",
	"0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9": "PairCreated(address,address,address,uint256)",
	"4c209b5fc8ad50758f13e2e1088ba56a560dff690a1c6fef26394f4c03821c4f": "Mint(address,uint256,uint256)",
	"dccd412f0b1252819cb1fd330b93224ca42612892bb3f4f789976e6d81936496": "Burn(address,uint256,uint256,address)",

	// Proxy Patterns (EIP-1967) - Common upgrade pattern
	"bc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b": "Upgraded(address)",
	"7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f": "AdminChanged(address,address)",
	"101b8081ff3b56bbf45deb824d86a3b0fd38b7e3dd42421105cf8abe9106db0b": "BeaconUpgraded(address)",

	// Bridge Events (Celer MessageBus: 0x9Bb46D5100d2Db4608112026951c9C965b233f4D, Wormhole)
	"1591690b8638f5fb2dbec82ac741805ac5da8b45dc5263f4875b0496fdce4e05": "TokensBridged(address,address,uint256,uint256,uint256)",

	// Oasis Network Specific Notes:
	// - Sapphire and Emerald ParaTimes are EVM-compatible and use standard EVM events
	// - wROSE (Wrapped ROSE) uses standard WETH9 events listed above
	// - YuzuSwap: First DEX on Emerald (Uniswap V2 fork) - 0x250d48C5E78f1E85F7AB07FEC61E93ba703aE668
	// - ValleySwap: DEX on Emerald - 0x7C0b0a525fc6A2caDf7AE37198119025C6feA28a
	// - LilacSwap: DEX on Emerald - 0x5ca7e4301C9ac05B94e4c5b0C1812015f0A0be5f
	// - Celer MessageBus: Cross-chain bridge - 0x9Bb46D5100d2Db4608112026951c9C965b233f4D
	// - Band Oracle: Price feeds on Sapphire - 0xDA7a001b254CD22e46d3eAB04d937489c93174C3
	// - Sapphire DeFi: Midas, Neby, Thorn, Accumulated Finance, BitProtocol
	//
	// To add more signatures, examine contract ABIs on:
	// - Sapphire: https://explorer.oasis.io/mainnet/sapphire
	// - Emerald: https://explorer.emerald.oasis.dev
}
