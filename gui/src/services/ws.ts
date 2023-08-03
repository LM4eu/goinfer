

const useWs = (
  onMsg: (data: any) => void,
  onStartInfer: () => void,
) => {
  const ws = new WebSocket('ws://localhost:5142/ws');

  ws.onopen = () => {
    console.log('WebSocket connected');
  };
  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    const msg = data.msg;
    // console.log("=>", data);
    if (data.nToken == 0) {
      onStartInfer();
    }
    if (data.nToken < 2) {
      if (msg == "\n") {
        return
      }
    }
    onMsg(msg);
    const resEl = document.getElementById("infer-block") as HTMLElement;
    resEl.scrollTop = resEl.scrollHeight;
  };
  ws.onerror = (event) => {
    console.log('WebSocket error', event);
  };
  ws.onclose = () => {
    console.log('WebSocket disconnected');
  };

  return { ws }
}

export { useWs }