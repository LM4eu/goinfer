import { api, models, mutateModel, stream, lmState } from "@/state";
import type { InferParams, InferResultContract, Task } from "@/interfaces";
import { ModelStateContract } from "@/interfaces";


async function infer(_prompt: string, _template: string, _params: InferParams): Promise<InferResultContract> {
  stream.value = "";
  lmState.isRunning = true;
  const res = await api.post<InferResultContract>("/infer", {
    "prompt": _prompt,
    "template": _template,
    ..._params,
  });
  if (res.ok) {
    console.log("RES", res)
  }
  lmState.isRunning = false;
  lmState.isStreaming = false;
  return res.data
}

async function abort() {
  const res = await api.get("/infer/abort");
  if (res.ok) {
    lmState.isRunning = false;
    lmState.isStreaming = false;
  }
}

async function loadModels() {
  const res = await api.get<ModelStateContract>("/model/state");
  console.log(JSON.stringify(res.data, null, "  "))
  if (res.ok) {
    models.splice(0, models.length, ...res.data.models);
    if (res.data.isModelLoaded) {
      mutateModel(res.data.loadedModel, res.data.ctx);
    }
  }
}

async function loadTasks(): Promise<Array<Record<string, any>>> {
  const res = await api.get<Array<Record<string, any>>>("/task/tree");
  console.log(JSON.stringify(res.data, null, "  "));
  if (res.ok) {
    return res.data
  }
  throw new Error("Error loading tasks")
}

async function loadTask(path: string) {
  const payload = {
    path: path
  }
  let task: Task = {
    name: "",
    model: "",
    template: "",
  };
  const res = await api.post<Record<string, any>>("/task/read", payload);
  if (res.ok) {
    //console.log("DATA", JSON.stringify(res.data, null, "  "));
    task = {
      name: res.data.name,
      model: res.data.model,
      template: res.data.template,
    }
    if ("modelConf" in res.data) {
      task.modelConf = res.data.modelConf;
    }
    if ("inferParams" in res.data) {
      task.inferParams = res.data.inferParams;
    }
  }
  return task
}

async function selectModel(name: string, ctx: number) {
  lmState.isLoadingModel = true;
  const res = await api.post("/model/load", { model: name, ctx: ctx });
  if (res.ok) {
    mutateModel(name, ctx);
  }
  lmState.isLoadingModel = false;
}

export { infer, abort, loadModels, selectModel, loadTasks, loadTask }