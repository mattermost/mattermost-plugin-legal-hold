import React from 'react';

import BaseSetting from './base_setting';

type Props = {
    id: string;
    name: string;
    helpText: string;
    onChange: (value: string) => void;
    value: string;
    disabled?: boolean;
};

const TextSetting = (props: Props) => {
    return (
        <BaseSetting
            {...props}
        >
            <input
                id={props.id}
                data-testid={`${props.id}-input`}
                className='form-control'
                type='text'
                value={props.value}
                onChange={(e) => props.onChange(e.target.value)}
                disabled={props.disabled}
            />
        </BaseSetting>
    );
};

export default TextSetting;
