import localForage from "localforage";
import { templates as templatesData } from "@/const/templates";
import { LmPrompt, LmTemplate } from "@/interfaces";

const useDb = () => {
  const prompts = localForage.createInstance({
    driver: localForage.INDEXEDDB,
    storeName: 'prompts',
  });
  const templates = localForage.createInstance({
    driver: localForage.INDEXEDDB,
    storeName: 'templates',
  });

  const init = async () => {
    if (await prompts.length() == 0) {
      console.log("The prompts db is empty")
    }
    if (await templates.length() == 0) {
      console.log("The templates db is empty, loading it with prebuilt templates");
      Object.values(templatesData).forEach((t) => {
        setTemplate(t.name, t.content)
      });
    }
  }

  const setPrompt = async (k: string, v: string) => {
    await prompts.ready();
    await prompts.setItem(k, v);
  };

  const delPrompt = async (k: string) => {
    await prompts.ready();
    await prompts.removeItem(k);
  };

  const loadPrompt = async (k: string): Promise<string> => {
    await prompts.ready();
    const v = await prompts.getItem<string>(k);
    if (!v) {
      throw new Error(`Key ${v} not found`)
    }
    return v
  };

  const listPromptsNames = async (): Promise<Array<string>> => {
    await prompts.ready();
    const _p = new Array<string>();
    await prompts.iterate((v, k, i) => {
      _p.push(k)
    });
    return _p
  };

  const setTemplate = async (k: string, v: string) => {
    await templates.ready();
    await templates.setItem(k, v);
  };

  const delTemplate = async (k: string) => {
    await templates.ready();
    await templates.removeItem(k);
  };

  const loadTemplate = async (k: string): Promise<LmTemplate> => {
    await templates.ready();
    const v = await templates.getItem<string>(k);
    if (!v) {
      throw new Error(`Key ${v} not found`)
    }
    return {
      name: k,
      content: v
    }
  };

  const listTemplatesNames = async (): Promise<Array<string>> => {
    await templates.ready();
    const _t = new Array<string>();
    await templates.iterate((v, k, i) => {
      _t.push(k)
    });
    return _t
  }

  return {
    init,
    setPrompt,
    delPrompt,
    loadPrompt,
    setTemplate,
    delTemplate,
    loadTemplate,
    listTemplatesNames,
    listPromptsNames,
  }
}

export { useDb }