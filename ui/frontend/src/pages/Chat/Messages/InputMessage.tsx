import React from 'react';
import classnames from 'classnames';
import production from 'react/jsx-runtime';
import remarkParse from 'remark-parse'
import remarkRehype from 'remark-rehype'
import rehypeHighlight from 'rehype-highlight';
import rehypeReact from "rehype-react";
import {unified} from 'unified';
import remarkGfm from 'remark-gfm';
import { EasyMessage, InputMessage, OutputMessage } from "@curaious/uno-converse";

interface IProps {
  message: InputMessage | EasyMessage | OutputMessage;
}

export const InputMessageRenderer: React.FC<IProps> = props => {
  const {message} = props;

  if (message.type === "message") {
    if (typeof message.content === "string") {
      const text = unified()
        .use(remarkParse)
        .use(remarkGfm)
        .use(remarkRehype as any)
        .use(rehypeHighlight as any)
        .use(rehypeReact, {
          ...production as any, components: {}
        }).processSync(message.content).result;

      return text as any;
    }

    if (Array.isArray(message.content) && message.content.length > 0) {
      return <>
        {message.content.map((item, i) => {
          if (item.type === "input_text" || item.type === "output_text") {
            const text = unified()
              .use(remarkParse)
              .use(remarkGfm)
              .use(remarkRehype as any)
              .use(rehypeHighlight as any)
              .use(rehypeReact, {
                ...production as any, components: {}
              }).processSync(item.text).result;

            return text as any;
          }

          if (item.type === "input_image") {
            return <img src={item.image_url} />
          }
        })}
      </>
    }
  }

  return null
}