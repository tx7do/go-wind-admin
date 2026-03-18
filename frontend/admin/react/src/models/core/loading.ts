import {useState} from 'react';

import type {ILoadingState} from '../types';

const initialState: ILoadingState = {
  isLoading: false,
  error: null,
};

/**
 * 加载状态管理 Model
 * 提供全局加载状态控制
 */
export default function LoadingModel() {
  const [state, setState] = useState<ILoadingState>(initialState);

  return {
    ...state,
    startLoading: () => setState({isLoading: true, error: null}),
    finishLoading: () => setState({isLoading: false, error: null}),
    loadingError: () => setState({isLoading: false, error: true}),
    clearError: () => setState({...state, error: null}),
    reset: () => setState(initialState),
  };
}
