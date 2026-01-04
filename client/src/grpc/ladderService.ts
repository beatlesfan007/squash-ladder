import {
  Player,
  ListPlayersRequest,
  ListPlayersResponse,
  AddPlayerRequest,
  AddPlayerResponse,
  AddMatchResultRequest,
  AddMatchResultResponse,
  SetScore,
  ListRecentMatchesRequest,
  ListRecentMatchesResponse,
  MatchResult,
  GenerateInviteRequest,
  GenerateInviteResponse,
  ClaimPlayerRequest,
  ClaimPlayerResponse,
} from './ladder_pb'
import { LadderServiceClient } from './ladder_grpc_web_pb'
import { supabase } from '../supabase'

export type {
  Player,
  ListPlayersRequest,
  ListPlayersResponse,
  MatchResult,
}

export {
  SetScore,
}

const client = new LadderServiceClient('/players')

const getMetadata = async () => {
  const { data: { session } } = await supabase.auth.getSession()
  const token = session?.access_token
  return token ? { 'authorization': `Bearer ${token}` } : {}
}

export const ladderService = {
  listPlayers: async (): Promise<ListPlayersResponse> => {
    const metadata = await getMetadata()
    return new Promise((resolve, reject) => {
      const request = new ListPlayersRequest()
      client.listPlayers(request, metadata as any, (err: any, response: any) => {
        if (err) {
          reject(new Error(`gRPC error: ${err.message || 'Unknown error'}`))
        } else if (response) {
          resolve(response)
        } else {
          reject(new Error('No response received'))
        }
      })
    })
  },

  addPlayer: async (name: string): Promise<{ player: Player, inviteToken: string }> => {
    const metadata = await getMetadata()
    return new Promise((resolve, reject) => {
      const request = new AddPlayerRequest()
      request.setName(name)

      client.addPlayer(request, metadata as any, (err: any, response: AddPlayerResponse) => {
        if (err) {
          reject(new Error(`gRPC error: ${err.message || 'Unknown error'}`))
        } else if (response) {
          const player = response.getPlayer()
          const inviteToken = response.getInviteToken()
          if (player) {
            resolve({ player, inviteToken })
          } else {
            reject(new Error('No player returned in response'))
          }
        } else {
          reject(new Error('No response received'))
        }
      })
    })
  },

  addMatchResult: async (
    challengerId: string,
    defenderId: string,
    winnerId: string,
    setScores: SetScore[]
  ): Promise<boolean> => {
    const metadata = await getMetadata()
    return new Promise((resolve, reject) => {
      const request = new AddMatchResultRequest()
      request.setChallengerId(challengerId)
      request.setDefenderId(defenderId)
      request.setWinnerId(winnerId)
      request.setSetScoresList(setScores)

      client.addMatchResult(request, metadata as any, (err: any, response: AddMatchResultResponse) => {
        if (err) {
          reject(new Error(`gRPC error: ${err.message || 'Unknown error'}`))
        } else if (response) {
          resolve(response.getSuccess())
        } else {
          reject(new Error('No response received'))
        }
      })
    })
  },

  listRecentMatches: async (limit: number): Promise<MatchResult[]> => {
    const metadata = await getMetadata()
    return new Promise((resolve, reject) => {
      const request = new ListRecentMatchesRequest()
      request.setLimit(limit)

      client.listRecentMatches(request, metadata as any, (err: any, response: ListRecentMatchesResponse) => {
        if (err) {
          reject(new Error(`gRPC error: ${err.message || 'Unknown error'}`))
        } else if (response) {
          resolve(response.getResultsList())
        } else {
          reject(new Error('No response received'))
        }
      })
    })
  },

  generateInvite: async (playerId: string): Promise<string> => {
    const metadata = await getMetadata()
    return new Promise((resolve, reject) => {
      const request = new GenerateInviteRequest()
      request.setPlayerId(playerId)
      client.generateInvite(request, metadata as any, (err: any, response: GenerateInviteResponse) => {
        if (err) reject(err)
        else resolve(response.getToken())
      })
    })
  },

  claimPlayer: async (token: string): Promise<ClaimPlayerResponse> => {
    const metadata = await getMetadata()
    return new Promise((resolve, reject) => {
      const request = new ClaimPlayerRequest()
      request.setToken(token)
      client.claimPlayer(request, metadata as any, (err: any, response: ClaimPlayerResponse) => {
        if (err) reject(err)
        else resolve(response)
      })
    })
  }
}
