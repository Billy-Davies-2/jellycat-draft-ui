declare module 'ws' {
  import { Duplex } from 'stream'
  export class WebSocketServer<T extends WebSocket = WebSocket> {
    constructor(opts: { port: number })
    on(event: string, cb: (...args: any[]) => void): void
    close(cb?: () => void): void
  }
  export class WebSocket extends Duplex {}
}
