```mermaid
graph TB
  %% --- ZONE D'INGESTION (Gestion de l'asynchronisme) ---
  subgraph INGEST["PIPELINE BIOTECH (Asynchrone)"]
    CLIENT["Jobs Pipeline<br/>(Input Variance: 12k - 128k)"]
    QUEUE[("Priority Queue<br/>Redis / Kafka<br/>Buffer: 30k RPM bursts")]
  end

  %% --- INFRASTRUCTURE SCALEWAY ---
  subgraph SCW["SCALEWAY VPC (k0s Cluster Bare Metal)"]
    
    %% ROUTING INTELLIGENT
    subgraph ROUTING["Inference Router"]
      SPLITTER["Python Router / Ingress<br/>Logic: Check Prompt Length"]
    end

    %% CONTROL PLANE (Séparé du GPU)
    subgraph CP["Control Plane (CPU Instances)"]
      K0S_CP["k0s Controllers<br/>Etcd + API Server"]
      KUBERAY["KubeRay Operator<br/>Orchestration RayCluster"]
    end

    %% STORAGE PARTAGÉ
    subgraph STORAGE["High Performance Storage"]
      NVME[("Local NVMe / HostPath<br/>Model Weights: 235GB")]
    end

    %% --- WORKER POOLS (TIERED ARCHITECTURE) ---
    
    %% POOL 1 : STANDARD LANE
    subgraph POOL_STD["POOL A: STANDARD LANE (< 32k tokens)"]
      HW_STD["Node: H100-SXM-4<br/>(4x H100 - 320GB VRAM)"]
      
      subgraph VLLM_STD["vLLM Engine (Standard)"]
        CONF_STD["Config:<br/>--tensor-parallel-size 4<br/>--kv-cache-dtype fp8<br/>Batch Size: High"]
      end
    end

    %% POOL 2 : HEAVY LANE
    subgraph POOL_HEAVY["POOL B: HEAVY LANE (> 32k tokens)"]
      HW_HEAVY["Node: H100-SXM-8<br/>(8x H100 - 640GB VRAM)"]
      
      subgraph VLLM_HEAVY["vLLM Engine (Heavy)"]
        CONF_HEAVY["Config:<br/>--tensor-parallel-size 8<br/>--enable-chunked-prefill<br/>--max-model-len 131072"]
      end
    end

  end

  %% --- FLUX DE DONNÉES ---
  CLIENT --> QUEUE
  QUEUE --> SPLITTER
  
  %% Routing Logic
  SPLITTER -->|Short/Medium Prompts<br/>Optimized for $/tok| VLLM_STD
  SPLITTER -->|Monster Prompts (128k)<br/>Optimized for Stability| VLLM_HEAVY

  %% Orchestration Links
  KUBERAY -.->|Manage Pods| VLLM_STD
  KUBERAY -.->|Manage Pods| VLLM_HEAVY
  K0S_CP --- HW_STD
  K0S_CP --- HW_HEAVY

  %% Storage Links
  NVME === VLLM_STD
  NVME === VLLM_HEAVY

  %% --- STYLING ---
  classDef scw fill:#2d0a4e,stroke:#4c1d95,stroke-width:3px,color:#fff
  classDef ingest fill:#1e293b,stroke:#94a3b8,stroke-width:2px,color:#fff
  classDef stdPool fill:#064e3b,stroke:#34d399,stroke-width:2px,color:#fff,stroke-dasharray: 5 5
  classDef heavyPool fill:#450a0a,stroke:#f87171,stroke-width:2px,color:#fff,stroke-dasharray: 5 5
  classDef logic fill:#172554,stroke:#60a5fa,stroke-width:2px,color:#fff
  classDef storage fill:#3f3f46,stroke:#a1a1aa,stroke-width:2px,color:#fff

  class SCW scw
  class INGEST ingest
  class POOL_STD stdPool
  class POOL_HEAVY heavyPool
  class ROUTING,CP logic
  class STORAGE storage
```
