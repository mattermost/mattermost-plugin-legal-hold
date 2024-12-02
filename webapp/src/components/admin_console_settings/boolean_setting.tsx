import React from 'react';

import BaseSetting from './base_setting';

type Props = {
    id: string;
    name: string;
    helpText: string;
    onChange: (value: boolean) => void;
    value: boolean;
    disabled?: boolean;
};

const BooleanSetting = (props: Props) => {
    return (
        <BaseSetting
            {...props}
        >
            <label className='radio-inline'>
                <input
                    data-testid={`${props.id}-true`}
                    type='radio'
                    onChange={() => props.onChange(true)}
                    checked={props.value}
                    disabled={props.disabled}
                />
                <span>{'true'}</span>
            </label>
            <label className='radio-inline'>
                <input
                    data-testid={`${props.id}-false`}
                    type='radio'
                    onChange={() => props.onChange(false)}
                    checked={!props.value}
                    disabled={props.disabled}
                />
                <span>{'false'}</span>
            </label>
        </BaseSetting>
    );
};

export default BooleanSetting;
