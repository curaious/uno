import React, {useCallback, useState} from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  applyEdgeChanges,
  applyNodeChanges,
  Node,
  Edge,
  EdgeChange, addEdge, Connection,
  ReactFlowProvider
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import {NodeChange} from "@xyflow/system";
import {AgentNode, Model, Tool} from "./Agent";
import { Drawer, Paper } from '@mui/material';

const initialNodes: Node[] = [
  {
    id: 'agent1',
    position: { x: 0, y: 0 },
    data: { label: 'Agent 1', id: 'agent1' },
    type: 'agent',
  },
  {
    id: 'model1',
    position: { x: 300, y: 200 },
    data: { label: 'Model 1', id: 'mode1' },
    type: 'model'
  },
  {
    id: 'tool1',
    position: { x: 100, y: 100 },
    data: { label: 'Tool 1', id: 'tool1' },
    type: 'tool',
  },
];

const initialEdges: Edge[] = [

];

export const Builder: React.FC = props => {
  const [nodes, setNodes] = useState(initialNodes);
  const [edges, setEdges] = useState(initialEdges);

  const onNodesChange = useCallback(
    (changes: NodeChange[]) => setNodes((nodesSnapshot) => applyNodeChanges(changes, nodesSnapshot)),
    [],
  );
  const onEdgesChange = useCallback(
    (changes: EdgeChange[]) => setEdges((edgesSnapshot) => applyEdgeChanges(changes, edgesSnapshot)),
    [],
  );
  const onConnect = useCallback(
    (params: Connection) => setEdges((edgesSnapshot) => addEdge(params, edgesSnapshot)),
    [],
  );

  return <div style={{ display: 'flex', flex: 1 }}>
    <div style={{ flex: 1 }}>
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onConnect={onConnect}
      nodeTypes={{
        "agent": AgentNode,
        "model": Model,
        "tool": Tool,
      }}
      colorMode="dark"
    >
      <Background />
      <Controls />
    </ReactFlow>
    </div>
    <Sidebar />
  </div>;
}

const Sidebar: React.FC = props => {
  const [open, setOpen] = useState(false);

  return <div style={{ width: 300 }}>
      <div></div>
    </div>
}