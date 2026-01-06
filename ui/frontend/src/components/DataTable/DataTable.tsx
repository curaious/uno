import React from 'react';
import styles from './DataTable.module.css';
import { Button } from '../shared/Buttons';

export interface Column<T> {
  key: keyof T;
  label: string;
  render?: (value: any, item: T) => React.ReactNode;
  width?: string;
}

export interface Action<T> {
  label: string;
  icon?: React.ReactNode;
  onClick: (item: T) => void;
  className?: string;
  title?: string;
}

export interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  actions?: Action<T>[];
  emptyState?: {
    icon: string | React.ReactNode;
    title: string;
    description: string;
    actionLabel: string;
    onAction: () => void;
  };
  loading?: boolean;
  error?: string | null;
  onRetry?: () => void;
}

export function DataTable<T extends { id: string }>({
  data,
  columns,
  actions = [],
  emptyState,
  loading = false,
  error = null,
  onRetry
}: DataTableProps<T>) {
  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>
          <div className={styles.spinner}></div>
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8.982 1.566a1.13 1.13 0 0 0-1.96 0L.165 13.233c-.457.778.091 1.767.98 1.767h13.713c.889 0 1.438-.99.98-1.767L8.982 1.566zM8 5c.535 0 .954.462.9.995l-.35 3.507a.552.552 0 0 1-1.1 0L7.1 5.995A.905.905 0 0 1 8 5zm.002 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2z"/>
          </svg>
          <div className={styles.errorContent}>
            <p>{error}</p>
            {onRetry && (
              <button className={styles.retryButton} onClick={onRetry}>
                Try Again
              </button>
            )}
          </div>
        </div>
      </div>
    );
  }

  if (data.length === 0 && emptyState) {
    return (
      <div className={styles.container}>
        <div className={styles.emptyState}>
          <div className={styles.emptyIcon}>{emptyState.icon}</div>
          <h3>{emptyState.title}</h3>
          <p>{emptyState.description}</p>
          <Button
            color="primary"
            variant="contained" onClick={emptyState.onAction}>
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
              <path d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
            </svg>
            {emptyState.actionLabel}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.tableContainer}>
      <table className={styles.table}>
        <thead>
          <tr>
            {columns.map((column) => (
              <th key={String(column.key)} style={{ width: column.width }}>
                {column.label}
              </th>
            ))}
            {actions.length > 0 && <th>Actions</th>}
          </tr>
        </thead>
        <tbody>
          {data.map((item) => (
            <tr key={item.id} className={styles.tableRow}>
              {columns.map((column) => (
                <td key={String(column.key)}>
                  {column.render 
                    ? column.render(item, item)
                    : String(item || '')
                  }
                </td>
              ))}
              {actions.length > 0 && (
                <td>
                  <div className={styles.actions}>
                    {actions.map((action, index) => (
                      <button
                        key={index}
                        className={`${styles.actionButton} ${action.className || ''}`}
                        onClick={() => action.onClick(item)}
                        title={action.title || action.label}
                      >
                        {action.icon}
                      </button>
                    ))}
                  </div>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
