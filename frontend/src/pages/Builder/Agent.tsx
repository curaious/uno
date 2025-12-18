import React from 'react';
import {Edge, Handle, NodeProps, Position, useEdges, useReactFlow, useStoreApi} from "@xyflow/react";
import { AgentIcon } from '../../Icons/AgentIcon';


type MyData = {
  label: string;
  incomingEdges: Edge[];
  outgoingEdges: Edge[];
};

export const AgentNode: React.FC<NodeProps> = props => {
  const { data } = props;
  const store = useStoreApi();
  const st = useReactFlow();

  console.log(data, store.getState(), st.getEdges())

  return <div style={{padding: '16px 64px', border: '1px solid #eee', background: '#1e1e1e', borderRadius: '4px', minWidth: '100px', display: 'inline-flex', justifyContent:'center'}}>
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}><AgentIcon style={{ height: 30, width: 30 }} /><span>AI Agent</span></div>
    <Handle id="model" type="target" position={Position.Bottom} isValidConnection={(connection) => {
      console.log(connection);
      return connection.sourceHandle === 'model'
    }}>
      <div style={{ position: 'absolute', top: 0, left: '-50%'}}>Model</div>
      <div style={{background: '#999', height: 8, width: 8, transform: 'translate(-50%, 6%)', 'rotate': '45deg'}}>
      </div>
      <div style={{ width: '1px', height: '40px', background: '#999', transform: 'translate(0, -2px)', position: 'relative'}}>
        
      </div>
      <div style={{ lineHeight: '14px', textAlign: 'center', border: '1px solid #999', borderRadius: 2, height: 16, width: 16, transform: 'translate(-50%, -14%)'}}>
        <strong style={{ color: '#999' }}>+</strong>
      </div>
    </Handle>
    <Handle id="tools" type="source" position={Position.Bottom} style={{ left: '25%'}}>
      <div style={{background: '#999', height: 8, width: 8, transform: 'translate(-50%, 6%)', 'rotate': '45deg'}}></div>
      <div style={{ width: '1px', height: '40px', background: '#999', transform: 'translate(0, -2px)'}}></div>
      <div style={{ lineHeight: '14px', textAlign: 'center', border: '1px solid #999', borderRadius: 2, height: 16, width: 16, transform: 'translate(-50%, -14%)'}}>
        <strong style={{ color: '#999' }}>+</strong>
      </div>
    </Handle>
    <Handle type="source" position={Position.Right}>
      <div style={{background: '#999', height: 8, width: 8, transform: 'translate(-35%, 13%)', 'rotate': '45deg'}}></div>
      <div style={{ width: '1px', height: '40px', background: '#999', rotate: '90deg', transform: 'translate(-25px, -60%)'}}></div>
      <div style={{ lineHeight: '14px', textAlign: 'center', border: '1px solid #999', borderRadius: 2, height: 16, width: 16, transform: 'translate(44px, -52px)'}}>
        <strong style={{ color: '#999' }}>+</strong>
      </div>
    </Handle>
    </div>;
}

export const Model: React.FC = props => {
  return <div style={{padding: '16px 64px', border: '1px solid #eee', background: '#1e1e1e', borderRadius: '50px', minWidth: '100px', display: 'inline-flex', justifyContent:'center'}}>
  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}><AgentIcon style={{ height: 30, width: 30 }} /><span>Model</span></div>
  <Handle id="model" type="source" position={Position.Top} isValidConnection={(connection) => {
    console.log(connection);
    return connection.sourceHandle === 'model'
  }}>
    <div style={{background: '#999', height: 8, width: 8, transform: 'translate(-50%, 6%)', 'rotate': '45deg'}}>
    </div>
    <div style={{ width: '1px', height: '40px', background: '#999', transform: 'translate(0, -50px)', position: 'relative'}}>
      
    </div>
    <div style={{ lineHeight: '14px', textAlign: 'center', border: '1px solid #999', borderRadius: 2, height: 16, width: 16, transform: 'translate(-50%, -106px)'}}>
      <strong style={{ color: '#999' }}>+</strong>
    </div>
  </Handle>
  </div>;
}

export const Tool: React.FC = props => {
  return <div style={{padding: '16px 64px', border: '1px solid #eee', background: '#1e1e1e', borderRadius: '4px', minWidth: '100px', display: 'inline-flex', justifyContent:'center'}}>
  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}><AgentIcon style={{ height: 30, width: 30 }} /><span>Tool</span></div>
  <Handle id="tool" type="source" position={Position.Top} isValidConnection={(connection) => {
    console.log(connection);
    return connection.sourceHandle === 'model'
  }}>
    <div style={{background: '#999', height: 8, width: 8, transform: 'translate(-50%, 6%)', 'rotate': '45deg'}}>
    </div>
    <div style={{ width: '1px', height: '40px', background: '#999', transform: 'translate(0, -50px)', position: 'relative'}}>
      
    </div>
    <div style={{ lineHeight: '14px', textAlign: 'center', border: '1px solid #999', borderRadius: 2, height: 16, width: 16, transform: 'translate(-50%, -106px)'}}>
      <strong style={{ color: '#999' }}>+</strong>
    </div>
  </Handle>
  </div>;
}